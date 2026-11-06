package controlplane

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

// The caller holds parserMutationMu. A repair is a new reviewed identity; the
// published origin and its runtime bundle remain unchanged while it is edited.
func (s *Server) cloneParserRepairDraft(w http.ResponseWriter, r *http.Request, p *middleware.SecurityContext, origin metastore.Document, body map[string]any) {
	var input struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Reason  string `json:"reason"`
	}
	if decodeJSON(r, &input, 8<<10) != nil || len(input.Name) > 256 || len(input.Reason) > 512 {
		jsonError(w, "repair draft request is invalid", http.StatusBadRequest)
		return
	}
	definition, ok := body["parser"].(map[string]any)
	if !ok || definition == nil {
		jsonError(w, "parser has no canonical editable definition; use compatibility repair", http.StatusConflict)
		return
	}
	encoded, err := json.Marshal(definition)
	if err != nil {
		jsonError(w, "source parser cannot be cloned", http.StatusConflict)
		return
	}
	parser := map[string]any{}
	if err := json.Unmarshal(encoded, &parser); err != nil {
		jsonError(w, "source parser cannot be cloned", http.StatusConflict)
		return
	}
	id := input.ID
	if id == "" {
		base := origin.ID
		if len(base) > 48 {
			base = base[:48]
		}
		id = fmt.Sprintf("%s-repair-%d", base, time.Now().UnixNano())
	}
	if safeID(id) != id || id == origin.ID {
		jsonError(w, "repair requires a distinct valid parser id", http.StatusBadRequest)
		return
	}
	if _, err := s.metadata.Get(r.Context(), "parser-packs", id, 0); err == nil {
		jsonError(w, "repair draft id already exists", http.StatusConflict)
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		jsonError(w, "repair draft uniqueness cannot be verified", http.StatusServiceUnavailable)
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name, _ = body["name"].(string)
		name += " repair"
		if len(name) > 256 {
			name = origin.ID + " repair"
		}
	}
	version := input.Version
	if version == "" {
		version, _ = body["version"].(string)
	}
	if version == "" {
		version = "0.1.0"
	}
	if !semverPattern.MatchString(version) {
		jsonError(w, "repair version must be semantic version", http.StatusBadRequest)
		return
	}
	if parser["pattern_type"] == "mapping" {
		priority, err := nfiMappingPriority(parser)
		if err != nil || priority >= 1000 {
			jsonError(w, "source mapping priority cannot be incremented; choose an explicit reviewed replacement strategy", http.StatusConflict)
			return
		}
		parser["mapping_priority"] = priority + 1
	}
	parser["id"] = id
	if unsafeDeclarative(parser) != nil {
		jsonError(w, "source parser is not a safe declarative definition", http.StatusConflict)
		return
	}
	checksum := parserArtifactChecksum(definition)
	payload := map[string]any{"id": id, "name": name, "version": version, "parser": parser, "cases": body["cases"], "artifact_checksum": parserArtifactChecksum(parser), "validation_passed": false, "validation": map[string]any{"status": "not_validated", "passed": 0, "total": 0}, "review_state": "draft", "activated": false, "repair_of": origin.ID, "repair_of_revision": origin.Revision, "repair_of_checksum": checksum, "repair_created_at": time.Now().UTC(), "repair_created_by": p.Username}
	if value, ok := body["source_quarantine_record_id"]; ok {
		payload["source_quarantine_record_id"] = value
	}
	data, err := json.Marshal(payload)
	if err != nil {
		jsonError(w, "repair draft encoding failed", http.StatusInternalServerError)
		return
	}
	saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: id, State: "draft", Payload: data, CreatedBy: p.Username})
	if err != nil {
		jsonError(w, "repair draft persistence failed", http.StatusInternalServerError)
		return
	}
	s.auditAction(p, "assurance.parser_clone_draft", "parser-packs/"+id, "success", map[string]string{"repair_of": origin.ID, "reason": strings.TrimSpace(input.Reason)})
	jsonCreated(w, map[string]any{"id": id, "revision": saved.Revision, "state": "draft", "repair_of": origin.ID, "repair_of_revision": origin.Revision, "original_active_preserved": true})
}
