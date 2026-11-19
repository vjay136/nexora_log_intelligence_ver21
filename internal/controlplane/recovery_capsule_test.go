package controlplane

import (
	"context"
	"encoding/json"
	"nexora.local/nli-v20/internal/recovery"
	metastore "nexora.local/nli-v20/internal/store"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedRecoveryCapsuleStoredAndDownloaded(t *testing.T) {
	f := newAPIFixture(t)
	config := t.TempDir()
	if err := os.WriteFile(filepath.Join(config, "controller.env"), []byte("private-recovery-value"), 0600); err != nil {
		t.Fatal(err)
	}
	contents, err := recovery.Capture(f.state, config)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := recovery.Seal(contents, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	response, raw := f.request("POST", "/api/v1/recovery/capsule", map[string]any{"capsule": json.RawMessage(encoded)})
	if response.StatusCode != 200 {
		t.Fatalf("import: %d %s", response.StatusCode, raw)
	}
	var result struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &result)
	got, err := f.server.readRecoveryCapsule(context.Background(), result.ID)
	if err != nil || string(got) != string(encoded) {
		t.Fatalf("download mismatch: %v", err)
	}
	opened, err := recovery.Open(got, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "state")
	cfg := filepath.Join(t.TempDir(), "config")
	if err := recovery.Restore(opened, state, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cfg, "controller.env"))
	if err != nil || string(data) != "private-recovery-value" {
		t.Fatal("protected configuration did not restore")
	}
}

func TestRecoveryCapsuleFreshControllerKeepsLoginAndPublishedCatalog(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "controller-state")
	config := filepath.Join(root, "configuration")
	web := filepath.Join(root, "web")
	if err := os.MkdirAll(web, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<title>Test</title>"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{StateDir: state, StorageDir: filepath.Join(state, "metadata"), WebDir: web, RecoveryConfigDir: config, EnrollmentToken: "Test-enrollment-token-1234567890", BootstrapAdminPassword: "Test-recovery-admin-password"}
	original, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := original.auth.Authenticate("admin", cfg.BootstrapAdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	_, err = original.metadata.Put(context.Background(), metastore.Document{Collection: "known-keys", ID: "recovered-key", State: "active", Payload: json.RawMessage(`{"name":"Recovered key","config":{"keys":["vjay"]}}`), CreatedBy: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := original.buildActiveParserBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := original.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "controller.json"), mustJSON(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	contents, err := recovery.Capture(state, config)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := recovery.Seal(contents, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := recovery.Open(encrypted, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(state); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(config); err != nil {
		t.Fatal(err)
	}
	if err := recovery.Restore(decoded, state, config); err != nil {
		t.Fatal(err)
	}
	cfg.BootstrapAdminPassword = ""
	replacement, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer replacement.Close()
	if username, _, err := replacement.auth.ValidateToken(token); err != nil || username != "admin" {
		t.Fatalf("controller login identity was lost: %s %v", username, err)
	}
	after, err := replacement.buildActiveParserBundle(context.Background())
	if err != nil || before.BundleHash != after.BundleHash {
		t.Fatalf("published catalog changed on host recovery: %s %s %v", before.BundleHash, after.BundleHash, err)
	}
	if status := replacement.audit.Verify(); status != nil {
		t.Fatalf("audit chain did not survive encrypted recovery: %+v", status)
	}
}
