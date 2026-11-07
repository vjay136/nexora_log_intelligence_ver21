package controlplane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

const maxInputPathRolloutTargets = 4096

func inputPathRequiresV22(inputs []any) bool {
	for _, value := range inputs {
		if input, ok := value.(map[string]any); ok && input["disabled"] == true {
			return true
		}
	}
	return len(inputs) == 0
}

func inputPathClientSupportsV22(version string) bool {
	major, _, _ := strings.Cut(strings.TrimPrefix(version, "v"), ".")
	value, err := strconv.Atoi(major)
	return err == nil && value >= 22
}

// handleInputPathProfileRollout proposes the published profile as a validated
// client config patch for one or more selected clients, or every eligible
// client. The client replaces only its inputs array after a separate approval.
func (s *Server) handleInputPathProfileRollout(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		jsonError(w, "GET or POST required", http.StatusMethodNotAllowed)
		return
	}
	permission := middleware.PermissionReadFleet
	if r.Method == http.MethodPost {
		permission = middleware.PermissionTriggerRollout
	}
	principal := s.authorize(w, r, permission)
	if principal == nil {
		return
	}
	doc, err := s.metadata.Get(r.Context(), "input-paths", id, 0)
	if err != nil || doc.Deleted {
		jsonError(w, "input path profile not found", http.StatusNotFound)
		return
	}
	if r.Method == http.MethodGet {
		s.listInputPathRollouts(w, r, id, doc)
		return
	}
	if doc.State != "active" {
		jsonError(w, "input path profile must be submitted, independently approved, and published before rollout", http.StatusConflict)
		return
	}
	var profile map[string]any
	if json.Unmarshal(doc.Payload, &profile) != nil {
		jsonError(w, "input path profile is invalid", http.StatusInternalServerError)
		return
	}
	config, _ := profile["config"].(map[string]any)
	inputs, ok := config["inputs"].([]any)
	if !ok || validateCatalogConfig("input-paths", config) != nil {
		jsonError(w, "input path profile failed validation", http.StatusConflict)
		return
	}
	var request struct {
		NodeIDs []string `json:"node_ids"`
	}
	if err := decodeJSON(r, &request, 1<<20); err != nil || len(request.NodeIDs) > maxInputPathRolloutTargets {
		jsonError(w, "invalid input path rollout request", http.StatusBadRequest)
		return
	}
	mode := "all_eligible"
	targets := []string{}
	requiresV22 := inputPathRequiresV22(inputs)
	if len(request.NodeIDs) > 0 {
		mode = "selected"
		known := map[string]*fleet.Node{}
		for _, node := range s.fleet.All() {
			known[node.ID] = node
		}
		seen := map[string]bool{}
		for _, nodeID := range request.NodeIDs {
			nodeID = strings.TrimSpace(nodeID)
			if nodeID == "" || seen[nodeID] {
				continue
			}
			node, exists := known[nodeID]
			if !exists {
				jsonError(w, "selected node does not exist: "+nodeID, http.StatusBadRequest)
				return
			}
			if requiresV22 && !inputPathClientSupportsV22(node.Version) {
				jsonError(w, "disabled paths and empty profiles require a Ver22 client: "+nodeID, http.StatusConflict)
				return
			}
			seen[nodeID] = true
			targets = append(targets, nodeID)
		}
	} else {
		for _, node := range s.fleet.All() {
			if node.Status != fleet.StatusOffline && (!requiresV22 || inputPathClientSupportsV22(node.Version)) {
				targets = append(targets, node.ID)
			}
		}
	}
	if len(targets) == 0 {
		jsonError(w, "no target clients are available", http.StatusConflict)
		return
	}
	if len(targets) > maxInputPathRolloutTargets {
		jsonError(w, "input path rollout exceeds the 4096-client target limit", http.StatusConflict)
		return
	}
	sort.Strings(targets)
	profileHash := fmt.Sprintf("%x", hashBytes(mustJSON(inputs)))
	proposal := map[string]any{
		"name":               "Roll out input paths: " + fmt.Sprint(profile["name"]),
		"type":               rolloutTypeConfigPush,
		"target_nodes":       targets,
		"payload":            map[string]any{"config_patch": map[string]any{"inputs": inputs}, "input_path_profile": map[string]any{"id": id, "revision": doc.Revision, "profile_hash": profileHash}},
		"input_path_profile": map[string]any{"id": id, "revision": doc.Revision, "name": profile["name"], "profile_hash": profileHash, "target_mode": mode, "requires_ver22": requiresV22},
		"approval_reason":    "Input path changes require independent rollout approval and client-side config validation.",
	}
	s.createRolloutProposal(w, r, principal, proposal)
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func (s *Server) listInputPathRollouts(w http.ResponseWriter, r *http.Request, profileID string, profile metastore.Document) {
	docs, err := s.metadata.List(r.Context(), "rollout-proposals", metastore.Query{Limit: 1000})
	if err != nil {
		jsonError(w, "input path rollout history unavailable", http.StatusServiceUnavailable)
		return
	}
	history := []map[string]any{}
	liveRollouts, inputStates := s.rollouts.inputPathSnapshot(true)
	for _, doc := range docs {
		var payload map[string]any
		if json.Unmarshal(doc.Payload, &payload) != nil {
			continue
		}
		pathProfile, _ := payload["input_path_profile"].(map[string]any)
		if fmt.Sprint(pathProfile["id"]) != profileID {
			continue
		}
		item := map[string]any{"id": doc.ID, "state": doc.State, "created_by": doc.CreatedBy, "created_at": doc.CreatedAt, "target_nodes": payload["target_nodes"], "profile": pathProfile}
		if liveID := strings.TrimSpace(fmt.Sprint(payload["live_rollout_id"])); liveID != "" && liveID != "<nil>" {
			if live := liveRollouts[liveID]; live != nil {
				item["rollout"] = live
			}
		}
		history = append(history, item)
	}
	sort.Slice(history, func(i, j int) bool {
		return fmt.Sprint(history[i]["created_at"]) > fmt.Sprint(history[j]["created_at"])
	})
	clients := []map[string]any{}
	for _, node := range s.fleet.All() {
		state := map[string]any{"node_id": node.ID, "status": "not_targeted"}
		current := inputStates[node.ID]
		desired, _ := current["desired_input_path_profile"].(map[string]any)
		applied, _ := current["applied_input_path_profile"].(map[string]any)
		if desired["id"] == profileID || applied["id"] == profileID {
			for key, value := range current {
				state[key] = value
			}
		}
		clients = append(clients, map[string]any{"node_id": node.ID, "hostname": node.Hostname, "status": node.Status, "input_path_sync": state})
	}
	jsonOK(w, map[string]any{"profile_id": profileID, "profile_revision": profile.Revision, "profile_state": profile.State, "history": history, "count": len(history), "clients": clients, "updated_at": time.Now().UTC()})
}

// Input status comes from the durable command results. Each client's desired
// status belongs to its newest command, and only completed results can
// establish an applied revision. This avoids a database request per client.
func (m *rolloutStateManager) inputPathSnapshot(includeHistory bool) (map[string]*rolloutRecord, map[string]map[string]any) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	type deployment struct {
		record  *rolloutRecord
		profile map[string]any
	}
	ordered := []deployment{}
	for _, item := range m.items {
		if item.Type != rolloutTypeConfigPush || item.StartedAt == nil {
			continue
		}
		var payload struct {
			Profile map[string]any `json:"input_path_profile"`
		}
		if json.Unmarshal(item.Payload, &payload) == nil && payload.Profile != nil {
			ordered = append(ordered, deployment{item, payload.Profile})
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].record.StartedAt.After(*ordered[j].record.StartedAt)
	})
	rollouts := make(map[string]*rolloutRecord, len(ordered))
	states := map[string]map[string]any{}
	for _, item := range ordered {
		live := item.record
		copy := *live
		if includeHistory {
			copy.NodeResults = make(map[string]*rolloutNodeResult, len(live.NodeResults))
		}
		for nodeID, result := range live.NodeResults {
			if includeHistory {
				resultCopy := *result
				copy.NodeResults[nodeID] = &resultCopy
			}
			state := states[nodeID]
			if state == nil {
				status := result.Status
				if status == rolloutStatusCompleted {
					status = "applied"
				} else if live.Status == rolloutStatusCancelled && status != rolloutStatusFailed {
					status = rolloutStatusCancelled
				}
				updatedAt := *live.StartedAt
				if !result.DoneAt.IsZero() {
					updatedAt = result.DoneAt
				}
				state = map[string]any{
					"node_id": nodeID, "desired_input_path_profile": item.profile,
					"input_path_sync_status": status, "input_path_sync_message": result.Message,
					"input_path_sync_updated_at": updatedAt,
				}
				states[nodeID] = state
			}
			if result.Status == rolloutStatusCompleted && state["applied_input_path_profile"] == nil {
				state["applied_input_path_profile"] = item.profile
			}
		}
		if includeHistory {
			rollouts[live.ID] = &copy
		}
	}
	return rollouts, states
}
