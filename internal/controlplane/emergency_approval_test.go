package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"nexora.local/nli-v20/internal/fleet"
	metastore "nexora.local/nli-v20/internal/store"
)

func TestAdminEmergencyParserSelfApprovalIsExplicitRevisionBoundAndAudited(t *testing.T) {
	fixture := newAPIFixture(t)
	doc, err := fixture.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "parser-packs", ID: "emergency-parser", State: "in_review",
		Payload: legacyParserPayload(t, "emergency-parser"), CreatedBy: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Merely being an administrator never bypasses separation of duties.
	response, body := fixture.request(http.MethodPost, "/api/v1/parser-packs/emergency-parser/approve", map[string]any{})
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("implicit self approval was accepted: %d %s", response.StatusCode, body)
	}

	fixture.server.cfg.AllowAdminEmergencySelfApproval = true
	response, body = fixture.request(http.MethodPost, "/api/v1/parser-packs/emergency-parser/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Production recovery requires this exact validated parser revision.",
		"expected_revision":       doc.Revision,
		"expected_checksum":       doc.Hash,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("explicit emergency parser approval failed: %d %s", response.StatusCode, body)
	}
	approved := decodeMap(t, body)
	payload := approved["payload"].(map[string]any)
	if approved["state"] != "approved" || payload["approval_mode"] != "emergency_admin_self" || payload["emergency_approved_by"] != "admin" {
		t.Fatalf("emergency approval evidence is incomplete: %s", body)
	}
}

func TestAdminEmergencyParserSelfApprovalRejectsStaleRevisionOrChecksum(t *testing.T) {
	fixture := newAPIFixture(t)
	fixture.server.cfg.AllowAdminEmergencySelfApproval = true
	doc, err := fixture.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "parser-packs", ID: "stale-emergency-parser", State: "in_review",
		Payload: legacyParserPayload(t, "stale-emergency-parser"), CreatedBy: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []map[string]any{
		{"emergency_self_approval": true, "reason": "Emergency recovery with stale revision evidence.", "expected_revision": doc.Revision + 1, "expected_checksum": doc.Hash},
		{"emergency_self_approval": true, "reason": "Emergency recovery with stale checksum evidence.", "expected_revision": doc.Revision, "expected_checksum": "stale"},
	} {
		response, body := fixture.request(http.MethodPost, "/api/v1/parser-packs/stale-emergency-parser/approve", request)
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("stale emergency evidence was accepted: %d %s", response.StatusCode, body)
		}
	}
}

func TestAdminEmergencyAssuranceSelfApprovalPublishesExactValidatedCandidate(t *testing.T) {
	fixture := newAPIFixture(t)
	fixture.server.cfg.AllowAdminEmergencySelfApproval = true
	parser := map[string]any{
		"pattern_type": "regex", "pattern": `^status=(?P<status>\w+)$`,
		"fields": []any{map[string]any{"name": "status", "source": "group:status", "type": "string"}},
	}
	cases := []any{map[string]any{"raw": "status=OK", "desired": map[string]any{"status": "OK"}}}
	response, body := fixture.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{
		"id": "emergency-assurance", "name": "Emergency assurance", "version": "1.0.0", "parser": parser, "cases": cases,
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create candidate failed: %d %s", response.StatusCode, body)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/emergency-assurance/submit", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("submit candidate failed: %d %s", response.StatusCode, body)
	}
	doc, err := fixture.server.metadata.Get(context.Background(), "parser-packs", "emergency-assurance", 0)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(doc.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/emergency-assurance/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Production quarantine recovery requires this exact validated candidate.",
		"expected_revision":       doc.Revision,
		"expected_checksum":       payload["artifact_checksum"],
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("emergency Assurance approval failed: %d %s", response.StatusCode, body)
	}
	approved, err := fixture.server.metadata.Get(context.Background(), "parser-packs", "emergency-assurance", 0)
	if err != nil {
		t.Fatal(err)
	}
	var approvedPayload map[string]any
	_ = json.Unmarshal(approved.Payload, &approvedPayload)
	if approved.State != "approved" || approvedPayload["approval_mode"] != "emergency_admin_self" || approvedPayload["approved_checksum"] != payload["artifact_checksum"] {
		t.Fatalf("Assurance emergency evidence is incomplete: %#v", approvedPayload)
	}
	response, body = fixture.request(http.MethodPost, "/api/v1/assurance/parsers/emergency-assurance/publish", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("approved Assurance candidate could not publish: %d %s", response.StatusCode, body)
	}
}

func TestAdminEmergencyParserRolloutSelfApprovalBindsExactBundle(t *testing.T) {
	fixture := newAPIFixture(t)
	fixture.server.cfg.AllowAdminEmergencySelfApproval = true
	fixture.server.fleet.Register(&fleet.Node{ID: "emergency-node", Hostname: "emergency-node"})
	if _, err := fixture.server.metadata.Put(context.Background(), metastore.Document{
		Collection: "regex-patterns", ID: "emergency-rule", State: "active",
		Payload: json.RawMessage(`{"name":"Emergency rule","pattern":"safe"}`), CreatedBy: "admin",
	}); err != nil {
		t.Fatal(err)
	}
	bundle, err := fixture.server.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := fixture.server.proposeParserBundleRollout(context.Background(), "admin", "Emergency parser rollout", "exact bundle recovery", "selected", bundle, []string{"emergency-node"})
	if err != nil {
		t.Fatal(err)
	}
	response, body := fixture.request(http.MethodPost, "/api/v1/rollouts/"+proposal.ID+"/approve", map[string]any{
		"emergency_self_approval": true,
		"reason":                  "Production parser recovery requires immediate exact-bundle rollout.",
		"expected_revision":       proposal.Revision,
		"expected_bundle_hash":    bundle.BundleHash,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("explicit emergency rollout approval failed: %d %s", response.StatusCode, body)
	}
	approved := decodeMap(t, body)
	payload := approved["payload"].(map[string]any)
	if payload["approval_mode"] != "emergency_admin_self" || payload["emergency_approved_by"] != "admin" || payload["live_rollout_id"] == "" {
		t.Fatalf("emergency rollout evidence is incomplete: %s", body)
	}
}
