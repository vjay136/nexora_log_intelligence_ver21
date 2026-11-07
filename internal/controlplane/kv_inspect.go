package controlplane

import (
	"errors"
	"net/http"

	"nexora.local/nli-v20/internal/fieldmapping"
	"nexora.local/nli-v20/internal/mixedkv"
)

// Inspection is intentionally read-only and uses the existing FieldSense
// admission budget. Runtime parsing never depends on this HTTP endpoint.
func (s *Server) handleKVInspect(w http.ResponseWriter, r *http.Request) {
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
		Raw   string             `json:"raw"`
		Field fieldmapping.Field `json:"field"`
	}
	if err := decodeJSON(r, &input, 2<<20); err != nil || len(input.Raw) == 0 || len(input.Raw) > fieldmapping.MaxInputBytes || input.Field.Method != "kv_all" {
		jsonError(w, "bounded raw evidence and a key/value tail field are required", http.StatusBadRequest)
		return
	}
	profiles, err := fieldmapping.InspectKeyValues(ctx, input.Raw, input.Field)
	admission := s.fieldSenseAnalysis()
	admission.kvInspections.Add(1)
	if len(profiles) == 0 {
		admission.kvNoMatch.Add(1)
	}
	if fieldSenseContextError(w, ctx) {
		return
	}
	result := map[string]any{"engine": "kvie-v1", "profiles": profiles, "persisted": false, "activated": false}
	if err != nil {
		result["diagnostic"] = err.Error()
		var diagnostic *mixedkv.Error
		if errors.As(err, &diagnostic) {
			result["diagnostic_code"] = diagnostic.Code
			result["diagnostic_offset"] = diagnostic.Offset
		}
	}
	jsonOK(w, result)
}
