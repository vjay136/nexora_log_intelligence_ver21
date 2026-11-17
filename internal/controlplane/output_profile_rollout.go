package controlplane

import (
	"encoding/json"
	"net/http"
	"slices"

	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/middleware"
)

func (s *Server) handleOutputProfileRollout(w http.ResponseWriter, r *http.Request, principal *middleware.SecurityContext, id string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var request map[string]any
	if decodeJSON(r, &request, 1<<20) != nil || request == nil {
		jsonError(w, "invalid endpoint rollout request", http.StatusBadRequest)
		return
	}
	scope, _ := request["scope"].(string)
	if scope == "" {
		jsonError(w, "endpoint rollout scope is required", http.StatusBadRequest)
		return
	}
	s.createRolloutProposal(w, r, principal, map[string]any{
		"type": rolloutTypeConfigPush, "name": "Endpoint profile rollout",
		"profile_id": id, "profile_revision": request["profile_revision"],
		"output_scope": scope, "target_nodes": request["target_nodes"],
	})
}

// bindOutputProfileProposal snapshots the exact active profile. The client
// supplies its intent and target, while the controller supplies the patch.
func (s *Server) bindOutputProfileProposal(w http.ResponseWriter, r *http.Request, proposal map[string]any) bool {
	scope, _ := proposal["output_scope"].(string)
	id, _ := proposal["profile_id"].(string)
	revision, _ := proposal["profile_revision"].(float64)
	if proposal["type"] != rolloutTypeConfigPush || id == "" || safeID(id) != id ||
		revision < 1 || (scope != "canary" && scope != "selected" && scope != "fleet") {
		jsonError(w, "an active endpoint profile revision and valid rollout scope are required", http.StatusBadRequest)
		return false
	}
	doc, err := s.metadata.Get(r.Context(), "output-destinations", id, 0)
	if err != nil || doc.Deleted || doc.State != "active" || float64(doc.Revision) != revision {
		jsonError(w, "endpoint profile revision is no longer active; reload and review the current revision", http.StatusConflict)
		return false
	}
	var profile struct {
		Name   string
		Config map[string]any
	}
	if json.Unmarshal(doc.Payload, &profile) != nil || validateCatalogConfig("output-destinations", profile.Config) != nil {
		jsonError(w, "published endpoint profile failed validation", http.StatusConflict)
		return false
	}
	requested := interfaceStrings(proposal["target_nodes"])
	if scope == "canary" && len(requested) != 1 || scope == "selected" && len(requested) == 0 || scope == "fleet" && len(requested) != 0 {
		jsonError(w, "choose exactly one canary client, selected clients, or the online fleet", http.StatusBadRequest)
		return false
	}
	online := make(map[string]bool)
	for _, node := range s.fleet.All() {
		if node.Status == fleet.StatusOnline || node.Status == fleet.StatusDegraded {
			online[node.ID] = true
		}
	}
	if scope == "fleet" {
		for id := range online {
			requested = append(requested, id)
		}
	}
	slices.Sort(requested)
	targets := slices.Compact(requested)
	if len(targets) == 0 || len(targets) > maxInputPathRolloutTargets {
		jsonError(w, "endpoint rollout requires 1 to 4096 online clients", http.StatusConflict)
		return false
	}
	for _, target := range targets {
		if !online[target] {
			jsonError(w, "endpoint rollout target is no longer online: "+target, http.StatusConflict)
			return false
		}
	}
	proposal["name"] = "Endpoint profile " + profile.Name + " (" + scope + ")"
	proposal["target_nodes"] = targets
	proposal["payload"] = map[string]any{"config_patch": map[string]any{"outputs": profile.Config["outputs"]}}
	return true
}
