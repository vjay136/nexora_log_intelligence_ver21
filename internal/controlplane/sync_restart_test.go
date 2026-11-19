package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"nexora.local/nli-v20/internal/engine"
	metastore "nexora.local/nli-v20/internal/store"
)

func TestCanonicalSyncIdentifiesControllerLifetime(t *testing.T) {
	first, next := NewStreamHub(2), NewStreamHub(2)
	before, after := first.CanonicalSnapshot(), next.CanonicalSnapshot()
	if before.ControllerInstanceID == "" || after.ControllerInstanceID == "" || before.ControllerInstanceID == after.ControllerInstanceID {
		t.Fatal("controller lifetimes must have distinct synchronization identities")
	}
	if again := first.CanonicalSnapshot(); again.ControllerInstanceID != before.ControllerInstanceID {
		t.Fatal("synchronization identity changed within one controller lifetime")
	}
	f := newAPIFixture(t)
	full := f.server.canonicalSyncSnapshot(context.Background())
	if full["controller_instance_id"] != f.server.streamTelemetry.CanonicalSnapshot().ControllerInstanceID {
		t.Fatal("full and stream snapshots must share the controller instance identity")
	}
}

func TestPublishedCatalogAndLoginSurviveControllerRestart(t *testing.T) {
	ctx := context.Background()
	cfg := Config{StateDir: t.TempDir(), WebDir: t.TempDir(), EnrollmentToken: "Persistent-test-enrollment-123456", BootstrapAdminPassword: "Restart-test-password-123!"}
	server, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if server != nil {
			_ = server.Close()
		}
	}()
	for collection, payload := range map[string]string{
		"templates":  `{"name":"Restart header","config":{"engine":"re2","pattern":"^(?P<event>restart) +","fields":{"event":"event"}}}`,
		"kv-config":  `{"name":"Restart pairs","config":{"pair_separators":[";"],"assignment_separators":["="],"literal_boundaries":true,"duplicate_policy":"error"}}`,
		"known-keys": `{"name":"Restart keys","config":{"keys":["status","reason"]}}`,
	} {
		if _, err := server.metadata.Put(ctx, metastore.Document{Collection: collection, ID: "restart-test", State: "active", Payload: json.RawMessage(payload), CreatedBy: "admin"}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := server.buildActiveParserBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := server.auth.Authenticate("admin", cfg.BootstrapAdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	instance := server.streamTelemetry.CanonicalSnapshot().ControllerInstanceID
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	server = nil
	cfg.BootstrapAdminPassword = ""
	server, err = New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := server.auth.ValidateToken(token); err != nil {
		t.Fatalf("persistent login identity lost: %v", err)
	}
	after, err := server.buildActiveParserBundle(ctx)
	if err != nil || before.BundleHash != after.BundleHash {
		t.Fatalf("published catalog changed across restart: %v", err)
	}
	if instance == server.streamTelemetry.CanonicalSnapshot().ControllerInstanceID {
		t.Fatal("controller restart reused its live sequence identity")
	}
	snapshot, err := server.automaticParserRuntime(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parsed := snapshot.parser.TryParseV5Production("restart status=failed;reason=not needed", engine.Source{Service: "auto"})
	if !parsed.Success || parsed.EventPayload.Fields["reason"] != "not needed" || parsed.EventPayload.Fields["status"] != "failed" {
		t.Fatalf("published template not restored: %+v", parsed)
	}
}
