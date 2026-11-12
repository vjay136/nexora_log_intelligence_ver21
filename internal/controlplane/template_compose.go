package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
)

const (
	maxTemplateComposeSamples = 500
	maxTemplateComposeDetails = 5
)

type templateComposeRef struct {
	ID       string `json:"id"`
	Revision int    `json:"revision"`
}

type templateComposeHeader struct {
	Engine  string            `json:"engine"`
	Pattern string            `json:"pattern"`
	Fields  map[string]string `json:"fields"`
}

func (s *Server) templateComposeConfig(ctx context.Context, collection string, ref templateComposeRef) (json.RawMessage, error) {
	if ref.Revision < 1 || ref.ID == "" || safeID(ref.ID) != ref.ID {
		return nil, fmt.Errorf("select an exact %s revision", collection)
	}
	doc, err := s.metadata.Get(ctx, collection, ref.ID, ref.Revision)
	if err != nil || doc.Deleted || doc.State == "revoked" {
		return nil, fmt.Errorf("selected %s revision is unavailable", collection)
	}
	var payload struct {
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(doc.Payload, &payload); err != nil || len(payload.Config) == 0 {
		return nil, fmt.Errorf("selected %s revision has no configuration", collection)
	}
	return payload.Config, nil
}

func composeTemplateMapping(header templateComposeHeader, kv mixedkv.Config) (fieldmapping.Plan, error) {
	plan := fieldmapping.Plan{Version: 1}
	if header.Engine != "" && header.Engine != "re2" || len(header.Pattern) > fieldmapping.MaxRegexBytes ||
		!strings.HasPrefix(header.Pattern, "^") || len(header.Fields) == 0 || len(header.Fields) >= fieldmapping.MaxFields {
		return plan, fmt.Errorf("header template needs an anchored RE2 prefix and named output fields")
	}
	re, err := regexp.Compile(header.Pattern)
	if err != nil || re.MatchString("") {
		return plan, fmt.Errorf("header pattern must be valid RE2 and consume a nonempty prefix")
	}
	groups := make([]string, 0, len(header.Fields))
	for group := range header.Fields {
		if re.SubexpIndex(group) < 1 {
			return plan, fmt.Errorf("header field refers to unknown named group %q", group)
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return re.SubexpIndex(groups[i]) < re.SubexpIndex(groups[j]) })
	for _, group := range groups {
		plan.Fields = append(plan.Fields, fieldmapping.Field{
			Name: header.Fields[group], Method: "regex", Source: header.Pattern, Part: group, Type: "string",
		})
	}
	if len(kv.KeyAllowPatterns) != 0 || len(kv.KeyDenyPatterns) != 0 || len(kv.StopKeys) != 0 || len(kv.ValueTypes) != 0 ||
		kv.AllowLeadingText || kv.MaxInputBytes != 0 && kv.MaxInputBytes != fieldmapping.MaxInputBytes ||
		kv.MaxKeyBytes != 0 && kv.MaxKeyBytes != 128 || kv.MaxValueBytes != 0 && kv.MaxValueBytes != fieldmapping.MaxInputBytes {
		return plan, fmt.Errorf("the combined template requires complete, string-valued KV coverage with standard input and value limits")
	}
	// Composition gives every configured separator literal boundary semantics.
	// An unexplained segment must fail or be explicitly captured by the asset.
	kv.LiteralBoundaries = true
	if _, err := mixedkv.Compile(kv); err != nil {
		return plan, err
	}
	if len(kv.PairSeparators) == 0 || len(kv.AssignmentSeparators) == 0 {
		return plan, fmt.Errorf("the KV template needs explicit pair and assignment separators")
	}
	policy := string(kv.DuplicatePolicy)
	if policy == "" || policy == "error" {
		policy = "reject"
	}
	tail := fieldmapping.Field{
		Name: "kv_pair", Type: "object", Method: "kv_all", Policy: "optional", KVProfile: "configured",
		KVHeaderPattern: header.Pattern, KVLiteralBoundaries: true, KVMaxPairs: kv.MaxFields,
		KVBareField: kv.BareField, KVEmptyKeyField: kv.EmptyKeyField, DuplicatePolicy: policy,
		KVPairSeparators: kv.PairSeparators, KVQuotes: kv.QuoteCharacters, KVEscape: kv.EscapeCharacter,
		KVKeyPattern: kv.KeyPattern, KVRejectEmpty: kv.RejectEmptyValues,
		KeyOptions: fieldmapping.KeyOptions{
			AssignmentSeparators: kv.AssignmentSeparators, KeyAliases: kv.KeyAliases, RequireComplete: true,
		},
	}
	if kv.BracketPairs != nil {
		tail.KVBrackets = &kv.BracketPairs
	}
	return fieldmapping.Plan{Version: 1, Fields: append(plan.Fields, tail)}, nil
}

func (s *Server) handleTemplateCompose(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	ctx, done, ok := s.beginFieldSenseAnalysis(w, r)
	if !ok {
		return
	}
	defer done()
	var input struct {
		Header  templateComposeRef `json:"header"`
		KV      templateComposeRef `json:"kv"`
		Samples []string           `json:"samples"`
	}
	if err := decodeJSON(r, &input, 2<<20); err != nil || len(input.Samples) == 0 || len(input.Samples) > maxTemplateComposeSamples {
		jsonError(w, "send 1 to 500 sample records with exact header and KV template revisions, up to 2 MiB total", http.StatusBadRequest)
		return
	}
	headerConfig, err := s.templateComposeConfig(ctx, "templates", input.Header)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	kvConfig, err := s.templateComposeConfig(ctx, "kv-config", input.KV)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	var header templateComposeHeader
	var kv mixedkv.Config
	decoder := json.NewDecoder(bytes.NewReader(kvConfig))
	decoder.DisallowUnknownFields()
	if json.Unmarshal(headerConfig, &header) != nil || decoder.Decode(&kv) != nil {
		jsonError(w, "template configuration is invalid or contains unsupported options", http.StatusBadRequest)
		return
	}
	plan, err := composeTemplateMapping(header, kv)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	compiled, err := fieldmapping.Compile(plan)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	examples, failures := []map[string]any{}, []map[string]any{}
	failed := 0
	for i, raw := range input.Samples {
		result, err := compiled.Execute(ctx, raw)
		if fieldSenseContextError(w, ctx) {
			return
		}
		if err == nil {
			err = compiled.ValidateComplete(raw, result)
		}
		if err != nil {
			failed++
			if len(failures) < maxTemplateComposeDetails {
				failures = append(failures, map[string]any{"line": i + 1, "error": err.Error()})
			}
		} else if len(examples) < maxTemplateComposeDetails {
			examples = append(examples, map[string]any{"line": i + 1, "actual": result.Values})
		}
	}
	jsonOK(w, map[string]any{
		"plan": plan, "scope_pattern": header.Pattern,
		"passed": failed == 0, "total": len(input.Samples), "failed": failed, "examples": examples, "failures": failures,
		"persisted": false, "activated": false,
	})
}
