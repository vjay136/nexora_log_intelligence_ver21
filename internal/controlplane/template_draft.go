package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"nexora.local/nli-v20/internal/mixedkv"
)

func (s *Server) handleTemplateDraft(w http.ResponseWriter, r *http.Request) {
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
		Kind    string         `json:"kind"`
		Config  map[string]any `json:"config"`
		Samples []string       `json:"samples"`
	}
	if err := decodeJSON(r, &input, 2<<20); err != nil || len(input.Samples) == 0 || len(input.Samples) > maxTemplateComposeSamples ||
		(input.Kind != "template" && input.Kind != "kv_template") {
		jsonError(w, "send 1 to 500 sample records, a template kind, and optional draft JSON, up to 2 MiB total", http.StatusBadRequest)
		return
	}
	if input.Config == nil {
		if input.Kind == "template" {
			input.Config = aiNativeMailAsset(input.Kind, input.Samples[0])
		} else {
			input.Config = suggestKVTemplate(ctx, input.Samples)
		}
		if fieldSenseContextError(w, ctx) {
			return
		}
		if input.Config == nil {
			jsonError(w, "No unambiguous template was found. Add representative records or enter and verify the JSON settings.", http.StatusUnprocessableEntity)
			return
		}
	}
	examples, failures := []map[string]any{}, []map[string]any{}
	failed := 0
	for i, sample := range input.Samples {
		var validation map[string]any
		var err error
		if strings.TrimSpace(sample) == "" || len(sample) > 1048576 || strings.ContainsAny(sample, "\r\n") {
			err = errors.New("each sample must be one nonempty log record up to 1 MiB")
		} else {
			validation, err = validateAIWorkspaceAsset(input.Kind, sample, input.Config)
		}
		if fieldSenseContextError(w, ctx) {
			return
		}
		if err != nil {
			failed++
			if len(failures) < maxTemplateComposeDetails {
				failures = append(failures, map[string]any{"line": i + 1, "error": err.Error()})
			}
		} else if len(examples) < maxTemplateComposeDetails {
			examples = append(examples, map[string]any{"line": i + 1, "actual": validation["actual"],
				"tail_start_byte": validation["tail_start_byte"], "tail_requires_kv_template": validation["tail_requires_kv_template"]})
		}
	}
	jsonOK(w, map[string]any{"passed": failed == 0, "config": input.Config, "total": len(input.Samples), "failed": failed,
		"examples": examples, "failures": failures, "persisted": false, "activated": false})
}

func suggestKVTemplate(ctx context.Context, samples []string) map[string]any {
	const maxInferenceSamples = 12
	samples = samples[:min(len(samples), maxInferenceSamples)]
	tails := make([]string, 0, len(samples))
	for _, sample := range samples {
		tail, _ := aiWorkspaceKVSample(sample)
		if strings.TrimSpace(tail) != "" {
			tails = append(tails, tail)
		}
	}
	var best mixedkv.Config
	var bestFields []map[string]any
	bestAssignments, bestBare := 0, 0
	ambiguous := false
	for _, separators := range [][]string{{";"}, {":"}, {"|"}, {","}, {";", ":"}, {";", "|"}, {":", "|"}, {",", "|"}, {",", ";"}, {":", ";", "|"}} {
		for _, assignments := range [][]string{{"="}, {":"}, {";"}, {"=", ":"}, {"=", ";"}, {"=", ":", ";"}} {
			for _, bare := range []string{"", "messages"} {
				if ctx.Err() != nil {
					return nil
				}
				cfg := mixedkv.Config{PairSeparators: separators, AssignmentSeparators: assignments,
					LiteralBoundaries: true, BareField: bare, MaxFields: 512, DuplicatePolicy: mixedkv.DuplicateError}
				parser, err := mixedkv.Compile(cfg)
				if err != nil {
					continue
				}
				fields, count, bareCount := scoreKVTemplate(parser, tails)
				if fields == nil || count < bestAssignments || count == bestAssignments && bareCount > bestBare {
					continue
				}
				switch {
				case count > bestAssignments || count == bestAssignments && bareCount < bestBare:
					best, bestFields, bestAssignments, bestBare, ambiguous = cfg, fields, count, bareCount, false
				case !reflect.DeepEqual(fields, bestFields):
					ambiguous = true
				}
			}
		}
	}
	if bestAssignments == 0 || ambiguous {
		return nil
	}
	var config map[string]any
	encoded, err := json.Marshal(best)
	if err != nil || json.Unmarshal(encoded, &config) != nil {
		return nil
	}
	return config
}

func scoreKVTemplate(parser *mixedkv.Parser, samples []string) ([]map[string]any, int, int) {
	fields := make([]map[string]any, 0, len(samples))
	assignments, bareCount := 0, 0
	for _, sample := range samples {
		result, err := parser.Parse(sample)
		if err != nil || result.RemainderSpan != nil || result.LeadingSpan != nil || len(result.Entries) == 0 {
			return nil, 0, 0
		}
		n := 0
		for _, entry := range result.Entries {
			if entry.AssignmentSeparator == "" {
				bareCount++
			} else {
				n++
			}
		}
		if n == 0 {
			return nil, 0, 0
		}
		fields, assignments = append(fields, result.Fields), assignments+n
	}
	return fields, assignments, bareCount
}
