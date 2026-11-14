package adapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"nexora.local/nli-v20/internal/canonicaljson"
	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
	parserpolicy "nexora.local/nli-v20/parser-policy"
	"nexora.local/nli-v20/telemetry"
)

const (
	maxRuntimeHeaderTemplates  = 128
	maxRuntimeKVTemplates      = 64
	maxRuntimeAssetBundleBytes = 16 << 20
)

type RuntimeTemplateAsset struct {
	AssetClass string          `json:"asset_class"`
	AssetID    string          `json:"asset_id"`
	Name       string          `json:"name"`
	Revision   int             `json:"revision"`
	SHA256     string          `json:"sha256"`
	Payload    json.RawMessage `json:"payload"`
}

type runtimeTemplateBundle struct {
	SchemaVersion int                    `json:"schema_version"`
	Assets        []RuntimeTemplateAsset `json:"assets"`
	BundleHash    string                 `json:"bundle_hash"`
}

type runtimeHeaderField struct {
	name     string
	index    int
	optional bool
}

type runtimeHeaderTemplate struct {
	id         string
	digest     string
	version    string
	pattern    *regexp.Regexp
	fields     []runtimeHeaderField
	timeout    time.Duration
	inputLimit int
}

type runtimeKVTemplate struct {
	id       string
	plan     *fieldmapping.Compiled
	fallback bool
}

type runtimeTemplates struct {
	headers []runtimeHeaderTemplate
	kv      []runtimeKVTemplate
}

var runtimeTemplateID = regexp.MustCompile("^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$")

// TemplateAssetsFromBundleFile reads the immutable generation written by client
// control. An absent bundle is valid for standalone and pre-enrollment clients.
func TemplateAssetsFromBundleFile(path string) ([]RuntimeTemplateAsset, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeAssetBundleBytes+1))
	if err != nil {
		return nil, err
	}
	return TemplateAssetsFromBundle(data)
}

// NewParserPackAdapterFromSynchronizedDirectories loads approved packages and
// templates from the same locally applied control-plane generation.
func NewParserPackAdapterFromSynchronizedDirectories(ctx context.Context, parserDir, assetDir string) (*ParserPackAdapter, error) {
	adapter, err := NewParserPackAdapterFromDirectory(parserDir)
	if err != nil || assetDir == "" {
		return adapter, err
	}
	assets, err := TemplateAssetsFromBundleFile(filepath.Join(assetDir, "bundle.json"))
	if err != nil {
		return nil, fmt.Errorf("load approved template generation: %w", err)
	}
	adapter, diagnostics, err := adapter.WithRuntimeTemplates(ctx, assets)
	for _, diagnostic := range diagnostics {
		log.Printf("WARNING: %s", diagnostic)
	}
	return adapter, err
}

func parserTemplateClass(class string) bool {
	return class == "templates" || class == "kv-config"
}

// TemplateAssetsFromBundle verifies the exact same digest client control checks
// before installing a synchronized generation.
func TemplateAssetsFromBundle(data []byte) ([]RuntimeTemplateAsset, error) {
	if len(data) == 0 || len(data) > maxRuntimeAssetBundleBytes {
		return nil, fmt.Errorf("invalid parser asset bundle size")
	}
	var bundle runtimeTemplateBundle
	if err := json.Unmarshal(data, &bundle); err != nil || bundle.SchemaVersion != 1 || len(bundle.Assets) > 10000 {
		return nil, fmt.Errorf("invalid parser asset bundle schema")
	}
	templates := []RuntimeTemplateAsset{}
	for i := range bundle.Assets {
		asset := &bundle.Assets[i]
		canonical, err := canonicaljson.Normalize(asset.Payload)
		if err != nil {
			return nil, fmt.Errorf("invalid canonical parser asset %s/%s: %w", asset.AssetClass, asset.AssetID, err)
		}
		digest := sha256.Sum256(canonical)
		if hex.EncodeToString(digest[:]) != asset.SHA256 {
			return nil, fmt.Errorf("invalid parser asset checksum %s/%s", asset.AssetClass, asset.AssetID)
		}
		asset.Payload = canonical
		if parserTemplateClass(asset.AssetClass) {
			templates = append(templates, *asset)
		}
	}
	unsigned, err := json.Marshal(struct {
		SchemaVersion int                    `json:"schema_version"`
		Assets        []RuntimeTemplateAsset `json:"assets"`
	}{bundle.SchemaVersion, bundle.Assets})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(unsigned)
	if hex.EncodeToString(digest[:]) != bundle.BundleHash {
		return nil, fmt.Errorf("parser asset bundle checksum mismatch")
	}
	return templates, nil
}

// WithRuntimeTemplates builds an independent adapter for the active template
// generation. Legacy catalog recipe shapes remain available in the UI and are
// isolated from execution when they cannot supply a complete parsing plan.
func (adapter *ParserPackAdapter) WithRuntimeTemplates(ctx context.Context, assets []RuntimeTemplateAsset) (*ParserPackAdapter, []string, error) {
	result := *adapter
	compiled := &runtimeTemplates{}
	diagnostics := []string{}
	diagnose := func(class, id string) {
		if len(diagnostics) < 32 {
			diagnostics = append(diagnostics, "Inactive in runtime: invalid or excess "+class+"/"+id)
		}
	}
	assets = append([]RuntimeTemplateAsset(nil), assets...)
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].AssetClass != assets[j].AssetClass {
			return assets[i].AssetClass < assets[j].AssetClass
		}
		return assets[i].AssetID < assets[j].AssetID
	})
	material := []string{adapter.policyHash}
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if !parserTemplateClass(asset.AssetClass) {
			continue
		}
		var envelope struct {
			Config json.RawMessage `json:"config"`
		}
		if !runtimeTemplateID.MatchString(asset.AssetID) || asset.Revision < 1 || json.Unmarshal(asset.Payload, &envelope) != nil || len(envelope.Config) == 0 {
			diagnose(asset.AssetClass, asset.AssetID)
			continue
		}
		switch asset.AssetClass {
		case "templates":
			if len(compiled.headers) >= maxRuntimeHeaderTemplates {
				diagnose(asset.AssetClass, asset.AssetID)
				continue
			}
			header, err := compileRuntimeHeader(asset, envelope.Config)
			if err != nil {
				diagnose(asset.AssetClass, asset.AssetID)
				continue
			}
			compiled.headers = append(compiled.headers, header)
		case "kv-config":
			if len(compiled.kv) >= maxRuntimeKVTemplates {
				diagnose(asset.AssetClass, asset.AssetID)
				continue
			}
			config := mixedkv.Config{}
			decoder := json.NewDecoder(bytes.NewReader(envelope.Config))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&config) != nil {
				diagnose(asset.AssetClass, asset.AssetID)
				continue
			}
			plan, err := compileRuntimeKV(&config)
			if err != nil {
				diagnose(asset.AssetClass, asset.AssetID)
				continue
			}
			compiled.kv = append(compiled.kv, runtimeKVTemplate{id: asset.AssetID, plan: plan})
		}
		digest := sha256.Sum256(asset.Payload)
		material = append(material, fmt.Sprintf("%s/%s@%d:%x", asset.AssetClass, asset.AssetID, asset.Revision, digest))
	}
	if len(compiled.headers) == 0 {
		result.templates = nil
		return &result, diagnostics, nil
	}
	fallback, err := compileRuntimeKV(nil)
	if err != nil {
		return nil, diagnostics, err
	}
	compiled.kv = append(compiled.kv, runtimeKVTemplate{id: fieldmapping.GlobalKVTemplateID, plan: fallback, fallback: true})
	digest := sha256.Sum256([]byte(strings.Join(material, "\x00")))
	result.templates, result.policyHash = compiled, hex.EncodeToString(digest[:])
	return &result, diagnostics, nil
}

func compileRuntimeHeader(asset RuntimeTemplateAsset, config json.RawMessage) (runtimeHeaderTemplate, error) {
	var header fieldmapping.HeaderTemplate
	decoder := json.NewDecoder(bytes.NewReader(config))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&header); err != nil {
		return runtimeHeaderTemplate{}, err
	}
	plan, err := fieldmapping.ComposeTemplates(header, nil)
	if err != nil {
		return runtimeHeaderTemplate{}, err
	}
	// Compile once to enforce the field-name, bounds, and mapping contract at
	// install time. The live header extraction only executes one RE2 match.
	if _, err = fieldmapping.Compile(plan); err != nil {
		return runtimeHeaderTemplate{}, err
	}
	re, err := regexp.Compile(header.Pattern)
	if err != nil {
		return runtimeHeaderTemplate{}, err
	}
	maxInput := header.ResourceLimits.MaxInputBytes
	timeout := time.Duration(header.ResourceLimits.TimeoutMS) * time.Millisecond
	if maxInput <= 0 || maxInput > fieldmapping.MaxInputBytes {
		maxInput = fieldmapping.MaxInputBytes
	}
	if timeout <= 0 || timeout > time.Second {
		timeout = 50 * time.Millisecond
	}
	fields := make([]runtimeHeaderField, 0, len(header.Fields))
	for _, field := range plan.Fields[:len(plan.Fields)-1] {
		fields = append(fields, runtimeHeaderField{name: field.Name, index: re.SubexpIndex(field.Part), optional: field.Policy == "optional"})
	}
	digest := sha256.Sum256(asset.Payload)
	return runtimeHeaderTemplate{id: "template/" + asset.AssetID, digest: hex.EncodeToString(digest[:]),
		version: fmt.Sprint(asset.Revision), pattern: re, fields: fields, timeout: timeout, inputLimit: maxInput}, nil
}

func compileRuntimeKV(config *mixedkv.Config) (*fieldmapping.Compiled, error) {
	const prefix = "prefix"
	plan, err := fieldmapping.ComposeTemplates(fieldmapping.HeaderTemplate{
		Engine: "re2", Pattern: "^(?P<prefix>synthetic-header) ", Fields: map[string]string{prefix: prefix},
	}, config)
	if err != nil {
		return nil, err
	}
	tail := plan.Fields[len(plan.Fields)-1]
	tail.KVHeaderPattern = ""
	return fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{tail}})
}

func extractRuntimeKV(ctx context.Context, profiles []runtimeKVTemplate, tail string) (fieldmapping.Result, string, error) {
	var result fieldmapping.Result
	used := ""
	for _, profile := range profiles {
		if err := ctx.Err(); err != nil {
			return fieldmapping.Result{}, "", err
		}
		if profile.fallback && used != "" {
			break
		}
		current, err := profile.plan.Execute(ctx, tail)
		if err != nil || len(current.Values) == 0 {
			continue
		}
		if used != "" && (!reflect.DeepEqual(result.Values, current.Values) || !reflect.DeepEqual(result.NestedEvidence, current.NestedEvidence)) {
			return fieldmapping.Result{}, "", fmt.Errorf("approved KV templates disagree on the extracted tail")
		}
		if used == "" {
			result, used = current, profile.id
		}
	}
	if used == "" {
		return fieldmapping.Result{}, "", fmt.Errorf("no complete KV template matched the tail")
	}
	return result, used, nil
}

type runtimeTemplateExecution struct {
	result    *parserpolicy.ExecutionResult
	end       int
	matched   bool
	ambiguous bool
	attempts  []telemetry.ParserAttempt
}

func (templates *runtimeTemplates) execute(ctx context.Context, raw, node string) runtimeTemplateExecution {
	var output runtimeTemplateExecution
	if templates == nil || len(raw) > fieldmapping.MaxInputBytes || !utf8.ValidString(raw) || strings.ContainsRune(raw, 0) {
		return output
	}
	for _, header := range templates.headers {
		if ctx.Err() != nil {
			break
		}
		if len(raw) > header.inputLimit {
			continue
		}
		started := time.Now()
		match := header.pattern.FindStringSubmatchIndex(raw)
		if len(match) == 0 || match[0] != 0 || match[1] == 0 {
			continue
		}
		output.matched = true
		child, cancel := context.WithTimeout(ctx, header.timeout)
		result, failure := header.execute(child, templates.kv, raw, match, node)
		cancel()
		attempt := telemetry.ParserAttempt{
			ParserID: header.id, Outcome: telemetry.AttemptOutcomeFailed,
			FailureReason: failure, LatencyMs: elapsedMillis(started),
		}
		if result != nil {
			attempt.Outcome = telemetry.AttemptOutcomeSuccess
			attempt.FieldsEmitted = result.EventPayload.FieldCount()
			attempt.StageResults = result.ParseTelemetry.OrderedAttempts[0].StageResults
			switch {
			case output.result == nil || match[1] > output.end:
				output.result, output.end, output.ambiguous = result, match[1], false
			case match[1] == output.end:
				if !reflect.DeepEqual(result.EventPayload.Fields, output.result.EventPayload.Fields) ||
					!reflect.DeepEqual(result.ParseTelemetry.FieldEvidence, output.result.ParseTelemetry.FieldEvidence) {
					output.ambiguous = true
				}
			}
		}
		output.attempts = append(output.attempts, attempt)
	}
	return output
}

func (header runtimeHeaderTemplate) execute(ctx context.Context, profiles []runtimeKVTemplate, raw string, match []int, node string) (*parserpolicy.ExecutionResult, string) {
	values := make(map[string]any, len(header.fields)+1)
	evidence := make(map[string]fieldmapping.Evidence, len(header.fields))
	telemetryEvidence := make([]telemetry.FieldEvidence, 0, len(header.fields)+1)
	for _, capture := range header.fields {
		start, end := match[capture.index*2], match[capture.index*2+1]
		if start < 0 || end <= start {
			if !capture.optional {
				return nil, "incomplete_structured_header"
			}
			continue
		}
		values[capture.name] = raw[start:end]
		evidence[capture.name] = fieldmapping.Evidence{Source: true, Start: start, End: end}
		telemetryEvidence = append(telemetryEvidence, telemetry.FieldEvidence{Field: capture.name, Source: true, Start: start, End: end})
	}
	if err := fieldmapping.ValidateMailHeader(raw, evidence); err != nil {
		return nil, "incomplete_structured_header"
	}
	suffix, offset, err := fieldmapping.TemplateTail(raw, match[1])
	if err != nil {
		return nil, "incomplete_structured_tail"
	}
	profile := ""
	if strings.TrimSpace(suffix) != "" {
		kv, kvProfile, err := extractRuntimeKV(ctx, profiles, suffix)
		if err != nil {
			return nil, "incomplete_structured_tail"
		}
		profile = kvProfile
		values["kv_pair"] = kv.Values["kv_pair"]
		parent := kv.Evidence["kv_pair"]
		telemetryEvidence = append(telemetryEvidence, telemetry.FieldEvidence{Field: "kv_pair", Source: true, Start: offset + parent.Start, End: offset + parent.End})
		keys := make([]string, 0, len(kv.NestedEvidence["kv_pair"]))
		for key := range kv.NestedEvidence["kv_pair"] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			evidence := kv.NestedEvidence["kv_pair"][key]
			spans := make([]telemetry.SourceSpan, len(evidence.Spans))
			for i, span := range evidence.Spans {
				spans[i] = telemetry.SourceSpan{Start: offset + span.Start, End: offset + span.End}
			}
			telemetryEvidence = append(telemetryEvidence, telemetry.FieldEvidence{Field: "kv_pair." + key, Source: true,
				Start: offset + evidence.Start, End: offset + evidence.End, Spans: spans})
		}
	}
	if ctx.Err() != nil || len(values) == 0 {
		return nil, "parser_execution_failed"
	}
	payload := &telemetry.EventPayload{Fields: values}
	if _, err := payload.MarshalJSON(); err != nil {
		return nil, "incomplete_structured_tail"
	}
	meta := telemetry.NewParseTelemetry(raw, node)
	meta.ParserID, meta.PackID, meta.ParserVersion, meta.PolicyHash = header.id, header.id, header.version, header.digest
	meta.TemplateID = header.id
	meta.Route, meta.Result, meta.Confidence, meta.EvidenceCoverage = "normal", "success", 1, 1
	meta.FieldEvidence = telemetryEvidence
	meta.InputFieldCount = len(values)
	meta.SourceFieldCount, meta.OutputFieldCount = len(values), len(values)
	stageName := "Published header template"
	if profile != "" {
		stageName += " + KV template " + profile
	}
	meta.StageCount = 1
	meta.OrderedAttempts = []telemetry.ParserAttempt{{ParserID: header.id, Outcome: telemetry.AttemptOutcomeSuccess,
		FieldsEmitted: len(values), StageResults: []telemetry.StageResult{{
			StageID: header.id, StageName: stageName, StageType: "mapping", Outcome: telemetry.StageOutcomeSuccess, FieldsAdded: len(values),
		}},
	}}
	return &parserpolicy.ExecutionResult{EventPayload: payload, ParseTelemetry: meta, ParserID: header.id}, ""
}
