package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	metastore "nexora.local/nli-v20/internal/store"
)

func TestAIProposalRejectAndRemoveLeavePublishedParserAndPoolUnchanged(t *testing.T) {
	f := newAPIFixture(t)
	definition, desired := nfiExampleDefinition()
	cases := []assuranceRequest{{Raw: nfiExample, Service: "smtp", Desired: desired}}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/parsers", map[string]any{"id": "proposal-control-parser", "name": "Reviewed parser", "version": "1.0.0", "parser": definition, "cases": cases})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("save: %d %s", response.StatusCode, body)
	}
	for _, action := range []string{"submit", "approve", "publish"} {
		response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/proposal-control-parser/"+action, map[string]any{"reason": "Administrator checked exact native output"})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
	ctx := context.Background()
	packBefore, err := f.server.metadata.Get(ctx, "parser-packs", "proposal-control-parser", 0)
	if err != nil {
		t.Fatal(err)
	}
	poolBefore, err := f.server.automaticParserRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"proposal": definition, "review_state": "draft", "source_hash": "synthetic-case-hash", "published": false})
	if _, err := f.server.metadata.Put(ctx, metastore.Document{Collection: "ai-proposals", ID: "proposal-controls", State: "draft", Payload: payload, CreatedBy: "admin"}); err != nil {
		t.Fatal(err)
	}
	response, _ = f.request(http.MethodPost, "/api/v1/assurance/ai-proposals/proposal-controls/reject", map[string]any{"reason": ""})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatal("empty rejection reason accepted")
	}
	response, body = f.request(http.MethodPost, "/api/v1/assurance/ai-proposals/proposal-controls/reject", map[string]any{"reason": "Use the existing approved parser instead"})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["parser_pack_changed"] != false {
		t.Fatalf("reject: %d %s", response.StatusCode, body)
	}
	response, _ = f.request(http.MethodPost, "/api/v1/assurance/ai-proposals/proposal-controls/approve", map[string]any{})
	if response.StatusCode != http.StatusConflict {
		t.Fatal("rejected proposal approved")
	}
	response, body = f.request(http.MethodDelete, "/api/v1/assurance/ai-proposals/proposal-controls", map[string]any{})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["parser_pack_changed"] != false {
		t.Fatalf("remove: %d %s", response.StatusCode, body)
	}
	response, _ = f.request(http.MethodGet, "/api/v1/assurance/ai-proposals/proposal-controls", nil)
	if response.StatusCode != http.StatusNotFound {
		t.Fatal("removed proposal remained in active queue")
	}
	packAfter, err := f.server.metadata.Get(ctx, "parser-packs", packBefore.ID, 0)
	if err != nil || packAfter.Revision != packBefore.Revision || packAfter.State != packBefore.State || string(packAfter.Payload) != string(packBefore.Payload) {
		t.Fatal("proposal controls modified a published parser")
	}
	poolAfter, err := f.server.automaticParserRuntime(ctx)
	if err != nil || poolAfter.hash != poolBefore.hash {
		t.Fatal("proposal removal changed the runtime pool")
	}
	if f.server.runAssurance(assuranceRequest{Raw: nfiExample, Service: "auto", Desired: desired})["passed"] != true {
		t.Fatal("published native parser stopped working")
	}
	// Revoking a derived pack is an explicit, separate governed operation.
	response, body = f.request(http.MethodPost, "/api/v1/assurance/parsers/proposal-control-parser/offboard", map[string]any{})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["approval_gate"] != "pending_approval" {
		t.Fatalf("explicit offboard did not prepare reviewed fleet removal: %d %s", response.StatusCode, body)
	}
	withdrawn, err := f.server.automaticParserRuntime(ctx)
	if err != nil || len(withdrawn.managedIDs) != 0 || len(withdrawn.builtinIDs) == 0 {
		t.Fatal("offboard retained the add-on or removed the embedded baseline")
	}
}

func TestAIProposalControlsRequireAuthorizedReviewerOrAuthor(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	if _, err := f.server.metadata.Put(ctx, metastore.Document{Collection: "ai-proposals", ID: "viewer-proposal", State: "draft", Payload: json.RawMessage(`{"review_state":"draft"}`), CreatedBy: "admin"}); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodPost, "/api/v1/users", map[string]any{"username": "proposal-viewer", "email": "viewer@example.test", "role": "viewer", "password": "Viewer-password-12345!"})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("viewer: %d %s", response.StatusCode, body)
	}
	f.login("proposal-viewer", "Viewer-password-12345!")
	f.refreshCSRF()
	for _, action := range []struct{ method, path string }{{http.MethodDelete, "/api/v1/assurance/ai-proposals/viewer-proposal"}, {http.MethodPost, "/api/v1/assurance/ai-proposals/viewer-proposal/reject"}} {
		response, _ = f.request(action.method, action.path, map[string]any{"reason": "Unauthorized removal"})
		if response.StatusCode != http.StatusForbidden {
			t.Fatalf("viewer modified proposal: %d", response.StatusCode)
		}
	}
	doc, err := f.server.metadata.Get(ctx, "ai-proposals", "viewer-proposal", 0)
	if err != nil || doc.State != "draft" || doc.Revision != 1 {
		t.Fatal("unauthorized proposal mutation")
	}
}
