package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	metastore "nexora.local/nli-v20/internal/store"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoveryCheckpointRetainsVerifiedCatalogAndStagesGovernedDraft(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	original := json.RawMessage(`{"name":"Recoverable keys","config":{"keys":["status","reason"]}}`)
	active, err := f.server.metadata.Put(ctx, metastore.Document{Collection: "known-keys", ID: "recovery-test", State: "active", Payload: original, CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := f.server.createRecoveryCheckpoint(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := f.server.createRecoveryCheckpoint(ctx, "admin")
	if err != nil || repeat.ID != checkpoint.ID {
		t.Fatalf("checkpoint idempotency: %s %v", repeat.ID, err)
	}
	export := filepath.Join(f.state, "recovery", "catalog.json")
	if _, err := os.Stat(export); err != nil {
		t.Fatal(err)
	}
	_, err = f.server.metadata.Put(ctx, metastore.Document{Collection: "known-keys", ID: active.ID, State: "active", Payload: json.RawMessage(`{"name":"Changed keys","config":{"keys":["status"]}}`), CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.server.buildActiveParserBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	report, err := f.server.stageRecoveryDrafts(ctx, checkpoint.ID, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(report["staged"].([]string)) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	draft, err := f.server.metadata.Get(ctx, "known-keys", active.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if draft.State != "draft" || priorActiveAssetRevision(draft) != 2 {
		t.Fatalf("missing active predecessor: %+v", draft)
	}
	after, err := f.server.buildActiveParserBundle(ctx)
	if err != nil || before.BundleHash != after.BundleHash {
		t.Fatal("recovery draft changed live bundle before approval")
	}
	again, err := f.server.stageRecoveryDrafts(ctx, checkpoint.ID, "admin")
	if err != nil || len(again["staged"].([]string)) != 1 {
		t.Fatalf("retry not idempotent: %+v %v", again, err)
	}
	response, data := f.request(http.MethodGet, "/api/v1/recovery", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status %d %s", response.StatusCode, data)
	}
	var status map[string]any
	_ = json.Unmarshal(data, &status)
	if len(status["checkpoints"].([]any)) != 1 {
		t.Fatalf("checkpoint status: %s", data)
	}
}

func TestRecoveryCheckpointRejectsTamperedArchive(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	_, err := f.server.metadata.Put(ctx, metastore.Document{Collection: "known-keys", ID: "recovery-test", State: "active", Payload: json.RawMessage(`{"name":"Keys","config":{"keys":["status"]}}`), CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := f.server.createRecoveryCheckpoint(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	manifest, _, err := f.server.readRecoveryCheckpoint(ctx, checkpoint.ID)
	if err != nil {
		t.Fatal(err)
	}
	ref := manifest.Documents[0]
	_, err = f.server.metadata.Put(ctx, metastore.Document{Collection: "recovery-documents", ID: ref.Archive, State: "archived", Payload: json.RawMessage(`{"collection":"known-keys","id":"other","revision":1}`), CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = f.server.readRecoveryCheckpoint(ctx, checkpoint.ID); err == nil {
		t.Fatal("tampered recovery archive accepted")
	}
}
