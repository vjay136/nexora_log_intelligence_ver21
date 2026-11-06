package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"nexora.local/nli-v20/internal/auth"
	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/middleware"
	"nexora.local/nli-v20/internal/nupire"
	metastore "nexora.local/nli-v20/internal/store"
)

func TestAdminApprovalUsesRoleRatherThanUsernameOrEmergencyPermission(t *testing.T) {
	doc := metastore.Document{CreatedBy: "tenant-administrator", Revision: 4}
	principal := &middleware.SecurityContext{Username: doc.CreatedBy, Role: auth.RoleAdmin}
	request := approvalRequest{Reason: "Administrator reviewed this change."}
	mode, details, err := (&Server{}).approvalMode(doc, principal, request, "digest", "parser_checksum")
	if err != nil || mode != "admin_self" || details["approved_revision"] != "4" {
		t.Fatalf("admin role self approval failed: mode=%q details=%v err=%v", mode, details, err)
	}
	principal.Role = auth.RoleReviewer
	principal.Permissions = []string{middleware.PermissionEmergencyApprove}
	if _, _, err := (&Server{}).approvalMode(doc, principal, request, "digest", "parser_checksum"); err == nil {
		t.Fatal("non-admin used an emergency permission to bypass separation of duties")
	}
}

func TestOwnChangeReviewPolicyRequiresAuthenticatedAdministrator(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		principal *middleware.SecurityContext
		want      bool
	}{
		{"unauthenticated", nil, false},
		{"empty identity", &middleware.SecurityContext{Role: auth.RoleAdmin}, false},
		{"admin role", &middleware.SecurityContext{Username: "creator", Role: auth.RoleAdmin}, true},
		{"reviewer creator", &middleware.SecurityContext{Username: "creator", Role: auth.RoleReviewer}, false},
		{"independent reviewer", &middleware.SecurityContext{Username: "reviewer", Role: auth.RoleReviewer}, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if got := principalCanReviewChange(scenario.principal, "creator"); got != scenario.want {
				t.Fatalf("own-change review policy = %v, want %v", got, scenario.want)
			}
		})
	}
}

func TestAdminCanRejectOwnAssuranceCandidate(t *testing.T) {
	f := newAPIFixture(t)
	if _, err := f.server.auth.CreateUser("admin", "reviewer-creator", "reviewer-creator@example.test", auth.RoleReviewer, "Review-password-12345!"); err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		actor, password string
		want            int
	}{
		{"admin", "Admin-password-12345!", http.StatusOK},
		{"reviewer-creator", "Review-password-12345!", http.StatusConflict},
	} {
		id := "reject-own-" + scenario.actor
		if _, err := f.server.metadata.Put(context.Background(), metastore.Document{
			Collection: "parser-packs", ID: id, State: "in_review",
			Payload: json.RawMessage(`{"review_state":"in_review"}`), CreatedBy: scenario.actor,
		}); err != nil {
			t.Fatal(err)
		}
		f.login(scenario.actor, scenario.password)
		f.refreshCSRF()
		response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers/"+id+"/reject", map[string]any{"reason": "Candidate needs corrected regression evidence."})
		if response.StatusCode != scenario.want {
			t.Fatalf("actor=%s: status=%d want=%d: %s", scenario.actor, response.StatusCode, scenario.want, body)
		}
		if scenario.want == http.StatusOK {
			doc, err := f.server.metadata.Get(context.Background(), "parser-packs", id, 0)
			if err != nil || doc.State != "rejected" {
				t.Fatalf("administrator rejection was not persisted: doc=%+v err=%v", doc, err)
			}
		}
	}
}

func TestAdminApprovalAcrossResourcesPreservesNonAdminSeparation(t *testing.T) {
	f := newAPIFixture(t)
	for _, account := range []struct{ username, role string }{{"tenant-administrator", auth.RoleAdmin}, {"global-reviewer", auth.RoleReviewer}} {
		if _, err := f.server.auth.CreateUser("admin", account.username, account.username+"@example.test", account.role, "Review-password-12345!"); err != nil {
			t.Fatal(err)
		}
	}
	f.server.fleet.Register(&fleet.Node{ID: "admin-approval-node", Hostname: "admin-approval-node"})
	for _, resource := range []struct {
		name, collection, state, path string
		payload                       json.RawMessage
	}{
		{"catalog", "parser-packs", "in_review", "/api/v1/parser-packs/", legacyParserPayload(t, "admin-approval-catalog")},
		{"asset", "regex-patterns", "in_review", "/api/v1/config/regex/", json.RawMessage(`{"name":"Review regex","pattern":"safe"}`)},
		{"ai", "ai-proposals", "draft", "/api/v1/assurance/ai-proposals/", json.RawMessage(`{"name":"Review proposal"}`)},
		{"rollout", "rollout-proposals", "pending_approval", "/api/v1/rollouts/", json.RawMessage(`{"name":"Review configuration","type":"config_push","target_nodes":["admin-approval-node"],"payload":{"config":{"logging":true}}}`)},
	} {
		t.Run(resource.name, func(t *testing.T) {
			for _, scenario := range []struct {
				actor, creator string
				want           int
			}{
				{"global-reviewer", "global-reviewer", http.StatusConflict},
				{"tenant-administrator", "tenant-administrator", http.StatusOK},
				{"tenant-administrator", "another-operator", http.StatusOK},
				{"global-reviewer", "another-operator", http.StatusOK},
			} {
				id := "admin-approval-" + resource.name
				if _, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: resource.collection, ID: id, State: resource.state, Payload: resource.payload, CreatedBy: scenario.creator}); err != nil {
					t.Fatal(err)
				}
				f.login(scenario.actor, "Review-password-12345!")
				f.refreshCSRF()
				response, body := f.request(http.MethodPost, resource.path+id+"/approve", map[string]any{"reason": "Administrator reviewed this exact change."})
				if response.StatusCode != scenario.want {
					t.Fatalf("actor=%s creator=%s: status=%d want=%d: %s", scenario.actor, scenario.creator, response.StatusCode, scenario.want, body)
				}
			}
		})
	}
}

func TestCatalogApprovalChecksumForAssuranceParserIsArtifactDigest(t *testing.T) {
	f := newAPIFixture(t)
	parser := map[string]any{
		"pattern_type": "regex", "pattern": `^status=(?P<status>\w+)$`,
		"fields": []any{map[string]any{"name": "status", "source": "group:status", "type": "string"}},
	}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{
		"id": "catalog-assurance-approval", "name": "Catalog assurance approval", "version": "1.0.0", "parser": parser,
		"cases": []any{map[string]any{"raw": "status=OK", "desired": map[string]any{"status": "OK"}}},
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/parser-packs/catalog-assurance-approval/submit", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("catalog submit: %d %s", response.StatusCode, body)
	}
	doc := decodeMap(t, body)
	digest := doc["approval_checksum"]
	if digest != doc["payload"].(map[string]any)["artifact_checksum"] || digest == doc["sha256"] {
		t.Fatalf("artifact and document approval checksums were conflated: %s", body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/parser-packs/catalog-assurance-approval/approve", map[string]any{
		"reason": "Administrator reviewed the exact catalog artifact.", "emergency_self_approval": true,
		"expected_revision": doc["revision"], "expected_checksum": digest,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("catalog approve: %d %s", response.StatusCode, body)
	}
	approved := decodeMap(t, body)["payload"].(map[string]any)
	if approved["approved_checksum"] != digest || approved["emergency_approved_by"] != "admin" {
		t.Fatalf("catalog approval lost artifact binding or attribution: %s", body)
	}
}

func TestAdminApprovalNUPIREPreservesEvidenceAndNonAdminSeparation(t *testing.T) {
	f := newAPIFixture(t)
	for _, account := range []struct{ username, role string }{{"evolution-admin", auth.RoleAdmin}, {"evolution-reviewer", auth.RoleReviewer}} {
		if _, err := f.server.auth.CreateUser("admin", account.username, account.username+"@example.test", account.role, "Review-password-12345!"); err != nil {
			t.Fatal(err)
		}
	}
	store := f.server.nupire.(persistentNUPIREController).store
	for _, actor := range []string{"evolution-admin", "evolution-reviewer"} {
		if _, err := store.CreateDiscovery(actor, nupire.Provenance{CreatedBy: actor, TrainingIDs: []string{"training"}, HeldOutIDs: heldOutIDs()}); err != nil {
			t.Fatal(err)
		}
		if _, err := store.SubmitCandidate(actor, json.RawMessage(`{"kind":"key_value"}`), actor); err != nil {
			t.Fatal(err)
		}
		f.login(actor, "Review-password-12345!")
		f.refreshCSRF()
		response, body := f.request(http.MethodPost, "/api/v1/nupire/candidates/"+actor+"/actions/approve", map[string]any{})
		if response.StatusCode != http.StatusConflict {
			t.Fatalf("approval bypassed evidence lifecycle: %d %s", response.StatusCode, body)
		}
		if _, err := store.Evaluate(actor, actor, nupire.Metrics{Samples: 10, ExactMatches: 10, FieldPrecision: 1, FieldRecall: 1}, heldOutIDs()); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Transition(actor, nupire.Shadow, actor, "Validated shadow evidence"); err != nil {
			t.Fatal(err)
		}
		response, body = f.request(http.MethodPost, "/api/v1/nupire/candidates/"+actor+"/actions/approve", map[string]any{})
		if actor == "evolution-admin" {
			if response.StatusCode != http.StatusOK || decodeMap(t, body)["approved_by"] != actor {
				t.Fatalf("admin self approval missing: %d %s", response.StatusCode, body)
			}
		} else if response.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("non-admin creator approved own evolution candidate: %d %s", response.StatusCode, body)
		}
	}
}
