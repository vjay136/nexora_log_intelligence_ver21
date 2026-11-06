package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"nexora.local/nli-v20/internal/adapter"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/familyprofile"
	"nexora.local/nli-v20/internal/fieldmapping"
	parserpolicy "nexora.local/nli-v20/parser-policy"
)

type automaticRuntimeSnapshot struct {
	hash        string
	parser      *adapter.ParserPackAdapter
	builtinIDs  []string
	managedIDs  []string
	diagnostics []string
}

type automaticContextParser struct {
	ctx    context.Context
	parser *adapter.ParserPackAdapter
}

func (p automaticContextParser) TryParseV5(raw string, source engine.Source) *engine.V5ParseResult {
	return p.parser.TryParseV5Context(p.ctx, raw, source)
}
func (p automaticContextParser) TryParseV5Raw(raw string, source engine.Source) *engine.V5ParseResult {
	return p.TryParseV5(raw, source)
}
func (p automaticContextParser) TryParseAutomaticV5Raw(raw string, source engine.Source) *engine.V5ParseResult {
	return p.TryParseV5(raw, source)
}

// Snapshot construction only reads canonical active metadata; no disk export,
// metadata update, approval or rollout is triggered by automatic analysis.
func (s *Server) automaticParserRuntime(ctx context.Context) (*automaticRuntimeSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.metadata == nil {
		return nil, fmt.Errorf("runtime metadata is not configured")
	}
	docs, err := s.listParserSyncDocuments(ctx, "parser-packs", "", parserSyncTargetLimit)
	if err != nil {
		return nil, err
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].ID < docs[j].ID })
	payloads := map[string][]byte{}
	seen := map[string][32]byte{}
	conflicts := map[string]bool{}
	diagnostics := []string{}
	total := 0
	diagnose := func(message string) {
		if len(diagnostics) < 32 {
			diagnostics = append(diagnostics, message)
		}
	}
	for _, doc := range docs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if doc.Deleted || (doc.State != "published" && doc.State != "active") {
			continue
		}
		pack, encoded, err := runtimeParserPackForDocumentContext(ctx, doc)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			diagnose("Invalid published parser isolated: " + doc.ID)
			continue
		}
		if familyprofile.IsBuiltinPackID(pack.ID) {
			diagnose("Protected embedded identity cannot be replaced: " + pack.ID)
			continue
		}
		digest := sha256.Sum256(encoded)
		if conflicts[pack.ID] {
			continue
		}
		if previous, exists := seen[pack.ID]; exists {
			if previous == digest {
				continue
			}
			conflicts[pack.ID] = true
			total -= len(payloads[pack.ID])
			delete(payloads, pack.ID)
			diagnose("Conflicting published runtime identity isolated: " + pack.ID)
			continue
		}
		seen[pack.ID] = digest
		if len(payloads) >= familyprofile.MaxManagedPacks || total+len(encoded) > familyprofile.MaxManagedBytes {
			diagnose("Managed runtime pool bound reached; excess parser isolated: " + pack.ID)
			continue
		}
		payloads[pack.ID] = encoded
		total += len(encoded)
	}
	names := make([]string, 0, len(payloads))
	for name := range payloads {
		names = append(names, name)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write(payloads[name])
		hash.Write([]byte{0})
	}
	poolHash := hex.EncodeToString(hash.Sum(nil))
	s.automaticRuntimeMu.Lock()
	cached := s.automaticRuntime
	s.automaticRuntimeMu.Unlock()
	if cached != nil && cached.hash == poolHash {
		if strings.Join(cached.diagnostics, "\x00") == strings.Join(diagnostics, "\x00") {
			return cached, nil
		}
		copyOfSnapshot := *cached
		copyOfSnapshot.diagnostics = diagnostics
		return &copyOfSnapshot, nil
	}
	parser, err := adapter.NewParserPackAdapterWithManagedPayloadsContext(ctx, payloads)
	if err != nil {
		return nil, err
	}
	builtins, err := familyprofile.Builtins()
	if err != nil {
		return nil, err
	}
	builtinIDs := []string{}
	for _, profile := range builtins {
		builtinIDs = append(builtinIDs, profile.Pack.ID)
	}
	compatID := parserpolicy.DefaultSMTPParserPack().ID
	present := false
	for _, id := range builtinIDs {
		if id == compatID {
			present = true
		}
	}
	if !present {
		builtinIDs = append(builtinIDs, compatID)
	}
	sort.Strings(builtinIDs)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	snapshot := &automaticRuntimeSnapshot{hash: poolHash, parser: parser, builtinIDs: builtinIDs, managedIDs: names, diagnostics: diagnostics}
	s.automaticRuntimeMu.Lock()
	s.automaticRuntime = snapshot
	s.automaticRuntimeMu.Unlock()
	return snapshot, nil
}

func (s *Server) automaticAssuranceDeps(ctx context.Context) (*engine.Deps, *automaticRuntimeSnapshot, error) {
	base := s.assuranceDeps
	if base == nil {
		base = engine.Default()
	}
	copyOfDeps := *base
	snapshot, err := s.automaticParserRuntime(ctx)
	if err != nil {
		// Never retain revoked managed candidates on a metadata failure.
		if builtin, ok := base.V5Parser.(*adapter.ParserPackAdapter); ok {
			copyOfDeps.V5Parser = automaticContextParser{ctx, builtin}
		}
		return &copyOfDeps, nil, err
	}
	copyOfDeps.V5Parser = automaticContextParser{ctx, snapshot.parser}
	return &copyOfDeps, snapshot, nil
}

func automaticRuntimeSummary(snapshot *automaticRuntimeSnapshot, err error) map[string]any {
	result := map[string]any{"available": err == nil && snapshot != nil, "builtin_profiles": []string{}, "managed_profiles": []string{}, "diagnostics": []string{}, "persisted": false}
	if snapshot != nil {
		result["pool_hash"] = snapshot.hash
		result["builtin_profiles"] = snapshot.builtinIDs
		result["managed_profiles"] = snapshot.managedIDs
		result["diagnostics"] = snapshot.diagnostics
	}
	if err != nil {
		result["diagnostics"] = []string{"Automatic managed runtime unavailable; embedded baseline preserved."}
	}
	return result
}

// Runtime names are separate proposals, never replacements for structural or
// operator-defined fields. A source-backed span must exactly reproduce value.
func automaticObservedFields(ctx context.Context, raw string, result *adapter.V5ParseResult) []fieldmapping.Field {
	fields := []fieldmapping.Field{}
	if result == nil || result.EventPayload == nil || result.ParseTelemetry == nil {
		return fields
	}
	used := map[string]bool{}
	for _, e := range result.ParseTelemetry.FieldEvidence {
		if ctx.Err() != nil || len(fields) >= fieldmapping.MaxFields {
			break
		}
		if !e.Source || e.Start < 0 || e.End <= e.Start || e.End > len(raw) || used[e.Field] || nfiSuggestionUniqueName(e.Field, map[string]bool{}) != e.Field {
			continue
		}
		value, exists := result.EventPayload.Fields[e.Field]
		if !exists || value == nil {
			continue
		}
		text := fmt.Sprint(value)
		if len(text) > 4096 || strings.Count(raw, text) != 1 {
			continue
		}
		kind := "string"
		switch value.(type) {
		case string:
		case bool:
			kind = "boolean"
		case int, int32, int64, uint, uint32, uint64:
			kind = "integer"
		case float32, float64, json.Number:
			kind = nfiSuggestedType(e.Field, text)
		default:
			continue
		}
		f := fieldmapping.Field{Name: e.Field, Label: strings.ReplaceAll(e.Field, "_", " "), Method: "byte", Source: strconv.Itoa(e.Start + 1), SourceEnd: strconv.Itoa(e.End), Type: kind, Policy: "required", Status: "suggested", Confidence: "observed approved runtime evidence", Evidence: "Approved runtime parser " + result.ParseTelemetry.ParserID + "; exact unique original-source bytes", Preview: text}
		compiled, err := fieldmapping.Compile(fieldmapping.Plan{Version: 1, Fields: []fieldmapping.Field{f}})
		if err != nil {
			continue
		}
		extracted, err := compiled.Execute(ctx, raw)
		if err != nil {
			continue
		}
		actual, _ := json.Marshal(extracted.Values[e.Field])
		expected, _ := json.Marshal(value)
		if string(actual) != string(expected) {
			continue
		}
		fields = append(fields, f)
		used[e.Field] = true
	}
	return fields
}

func (s *Server) automaticSuggestionEvidence(ctx context.Context, samples []string, service string) (map[string]any, []fieldmapping.Field) {
	snapshot, err := s.automaticParserRuntime(ctx)
	info := automaticRuntimeSummary(snapshot, err)
	matches := []map[string]any{}
	proposals := []fieldmapping.Field{}
	info["matches"] = matches
	if err != nil {
		return info, proposals
	}
	service = strings.TrimSpace(service)
	if service == "" || service == "*" || strings.Contains(service, ",") {
		service = "auto"
	}
	const evidenceLimit = 64 << 10
	for index, raw := range samples {
		if index >= 16 || ctx.Err() != nil {
			break
		}
		result := snapshot.parser.TryParseV5ProductionContext(ctx, raw, engine.Source{Hostname: "fieldsense-analysis", Service: service})
		if result == nil {
			continue
		}
		match := map[string]any{"sample_index": index, "success": result.Success, "fields": []map[string]any{}, "attempts": []map[string]any{}, "route": "unmatched", "parser_id": "", "attempt_count": 0}
		if telemetry := result.ParseTelemetry; telemetry != nil {
			match["parser_id"] = telemetry.ParserID
			match["attempt_count"] = telemetry.AttemptCount
			match["route"] = telemetry.Route
			attempts := []map[string]any{}
			for i, attempt := range telemetry.OrderedAttempts {
				if i >= 32 {
					break
				}
				attempts = append(attempts, map[string]any{"parser_id": attempt.ParserID, "outcome": attempt.Outcome, "fields": attempt.FieldsEmitted})
			}
			match["attempts"] = attempts
			fields := []map[string]any{}
			for _, e := range telemetry.FieldEvidence {
				if len(fields) >= 32 || result.EventPayload == nil {
					break
				}
				value, exists := result.EventPayload.Fields[e.Field]
				if !exists {
					continue
				}
				text := []rune(fmt.Sprint(value))
				if len(text) > 128 {
					text = text[:128]
				}
				fields = append(fields, map[string]any{"name": e.Field, "preview": string(text), "source_backed": e.Source, "start": e.Start, "end": e.End})
			}
			match["fields"] = fields
		}
		candidate := append(matches, match)
		info["matches"] = candidate
		encoded, _ := json.Marshal(info)
		if len(encoded) > evidenceLimit-1024 {
			break
		}
		matches = candidate
		if index == 0 {
			proposals = automaticObservedFields(ctx, raw, result)
		}
	}
	info["matches"] = matches
	info["checked_samples"] = len(matches)
	if len(matches) < len(samples) {
		info["diagnostics"] = append(append([]string{}, snapshot.diagnostics...), "Runtime evidence is bounded to 16 samples and 64 KiB; structural corpus suggestions remain separate.")
	}
	return info, proposals
}
