package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"nexora.local/nli-v20/internal/fleet"
	metastore "nexora.local/nli-v20/internal/store"
)

func TestOutputProfileRolloutPinsPublishedRouteAndCanary(t *testing.T) {
	f := newAPIFixture(t)
	for _, id := range []string{"endpoint-test-b", "endpoint-test-a"} {
		f.server.fleet.Register(&fleet.Node{ID: id, Hostname: id})
	}
	profile, err := f.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "output-destinations", ID: "endpoint-test", State: "active", CreatedBy: "admin",
		Payload: json.RawMessage(`{"name":"Endpoint test","config":{"outputs":[{"type":"loki","url":"https://loki.example.test","fallback_path":"/var/log/nli/loki-failures.jsonl","routes":["normal","warning"]}]}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name     string
		scope    string
		targets  []string
		revision int
		status   int
		want     []any
	}{
		{"one canary", "canary", []string{"endpoint-test-a"}, profile.Revision, http.StatusCreated, []any{"endpoint-test-a"}},
		{"reject multiple canary clients", "canary", []string{"endpoint-test-a", "endpoint-test-b"}, profile.Revision, http.StatusBadRequest, nil},
		{"select online fleet", "fleet", nil, profile.Revision, http.StatusCreated, []any{"endpoint-test-a", "endpoint-test-b"}},
		{"reject unlisted target", "selected", []string{"unknown"}, profile.Revision, http.StatusConflict, nil},
		{"reject stale revision", "canary", []string{"endpoint-test-a"}, profile.Revision + 1, http.StatusConflict, nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			resp, body := f.request(http.MethodPost, "/api/v1/config/output-destinations/"+profile.ID+"/rollout", map[string]any{
				"profile_revision": scenario.revision, "target_nodes": scenario.targets, "scope": scenario.scope,
				"payload": map[string]any{"config_patch": map[string]any{"outputs": []any{}, "inputs": []any{}}},
			})
			if resp.StatusCode != scenario.status {
				t.Fatalf("status %d, want %d: %s", resp.StatusCode, scenario.status, body)
			}
			if scenario.status != http.StatusCreated {
				return
			}
			item := decodeMap(t, body)
			payload := item["payload"].(map[string]any)
			if item["state"] != "pending_approval" || !reflect.DeepEqual(payload["target_nodes"], scenario.want) {
				t.Fatalf("unexpected proposal: %s", body)
			}
			patch := payload["payload"].(map[string]any)["config_patch"].(map[string]any)
			outputs := patch["outputs"].([]any)
			if len(patch) != 1 || len(outputs) != 1 || outputs[0].(map[string]any)["url"] != "https://loki.example.test" {
				t.Fatalf("proposal did not bind the canonical outputs-only profile: %s", body)
			}
		})
	}
}
