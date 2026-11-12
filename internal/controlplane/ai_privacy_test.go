package controlplane

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"nexora.local/nli-v20/internal/store"
)

func TestAIPrivacyProductionEvidenceStaysLocal(t *testing.T) {
	f := newAPIFixture(t)
	calls := mappingAIProviderFixture(t, f, "No external request was expected", nil)
	const raw = "status=failed reason=retained-private-sentinel"
	for _, tc := range []struct {
		name string
		path string
		body map[string]any
		want int
	}{
		{"mapping", "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": raw, "evidence_source": "production"}, http.StatusOK},
		{"workspace mapping", "/api/v1/ai/workspace", map[string]any{"raw": raw, "workspace": "quarantine-wizard", "mode": "fix", "artifact": "parser", "evidence_source": "production"}, http.StatusOK},
		{"workspace guidance", "/api/v1/ai/workspace", map[string]any{"raw": raw, "workspace": "quarantine-wizard", "mode": "ask", "evidence_source": "production"}, http.StatusUnprocessableEntity},
		{"assurance", "/api/v1/assurance/ai-proposal", map[string]any{"raw": raw, "source": "quarantine:incident-001"}, http.StatusUnprocessableEntity},
		{"unknown classification", "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": raw, "evidence_source": "other"}, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, body := f.request(http.MethodPost, tc.path, tc.body)
			if response.StatusCode != tc.want {
				t.Fatalf("HTTP %d, want %d: %s", response.StatusCode, tc.want, body)
			}
			if tc.want == http.StatusOK {
				result := decodeMap(t, body)
				actual := result["validation"].(map[string]any)["actual"].(map[string]any)
				if result["proposal_origin"] != "native_fallback" || result["provider_attempts"] != float64(0) || actual["status"] != "failed" {
					t.Fatalf("mapping must provide verified native values with zero provider attempts: %s", body)
				}
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("production evidence reached provider %d times", calls.Load())
	}

	response, body := f.request(http.MethodGet, "/api/v1/ai/privacy", nil)
	if response.StatusCode != http.StatusOK || bytes.Contains(body, []byte("retained-private-sentinel")) {
		t.Fatalf("privacy monitor failed or included a raw value: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	activity := result["activity"].(map[string]any)
	if activity["production_withheld"] != float64(4) || len(activity["events"].([]any)) != 4 {
		t.Fatalf("missing production evidence decisions: %v", activity)
	}
}

func TestAIPrivacyKnowledgeReturnsInventoryWithoutContent(t *testing.T) {
	f := newAPIFixture(t)
	if _, err := f.server.metadata.Put(context.Background(), store.Document{
		Collection: "known-keys", ID: "private-catalog-name", State: "active", Payload: []byte(`{"config":{"keys":["private-key-example"]}}`),
	}); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodGet, "/api/v1/ai/privacy", nil)
	if response.StatusCode != http.StatusOK || bytes.Contains(body, []byte("private-")) {
		t.Fatalf("knowledge response exposed catalog content: %d %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	knowledge := result["knowledge"].(map[string]any)
	workspaces := knowledge["workspaces"].([]any)
	if len(workspaces) != len(aiWorkspaceWorkflows) {
		t.Fatalf("knowledge has %d workspaces, want %d", len(workspaces), len(aiWorkspaceWorkflows))
	}
	count := knowledge["inventory"].(map[string]any)["known_key_catalogs"].(map[string]any)
	if count["available"] != true || count["count"].(float64) < 1 {
		t.Fatalf("missing configured catalog count: %v", count)
	}
}

func TestAIPrivacyHistoryIsBoundedAndNewestFirst(t *testing.T) {
	var privacy aiPrivacyStore
	for i := 0; i < aiPrivacyHistorySize; i++ {
		privacy.record("workspace", aiPrivacyMasked)
	}
	privacy.record("mapping", aiPrivacyWithheld)
	snapshot := privacy.snapshot()
	events := snapshot["events"].([]aiPrivacyEvent)
	if len(events) != aiPrivacyHistorySize || events[0].Route != "mapping" || snapshot[aiPrivacyMasked] != uint64(aiPrivacyHistorySize) {
		t.Fatalf("unexpected audit history: %v", snapshot)
	}
}
