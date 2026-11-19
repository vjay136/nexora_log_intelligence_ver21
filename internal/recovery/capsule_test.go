package recovery

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCapsuleEncryptRestoreAndRejectInvalidContents(t *testing.T) {
	root := t.TempDir()
	state := filepath.Join(root, "state")
	config := filepath.Join(root, "config")
	files := map[string][]byte{filepath.Join(state, "users.json"): []byte(`{"users":[{"username":"admin"}]}`), filepath.Join(state, "tls", "ca.key"): []byte("private-ca-identity"), filepath.Join(state, "tls", "server.key"): []byte("private-server-key"), filepath.Join(config, "controller.env"): []byte("SUPABASE_SECRET_KEY=private-value"), filepath.Join(state, "metadata", "known-keys", "key.json"): []byte(`{"config":{"keys":["vjay"]}}`)}
	for path, data := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	contents, err := Capture(state, config)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := Seal(contents, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("private-ca-identity")) || bytes.Contains(encoded, []byte("private-value")) {
		t.Fatal("identity leaked from encrypted capsule")
	}
	if _, err := Open(encoded, "Incorrect-passphrase-12345"); err == nil {
		t.Fatal("wrong recovery password accepted")
	}
	opened, err := Open(encoded, "Test-recovery-passphrase-123")
	if err != nil {
		t.Fatal(err)
	}
	restoredState := filepath.Join(root, "restored-state")
	restoredConfig := filepath.Join(root, "restored-config")
	if err := Restore(opened, restoredState, restoredConfig); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(restoredState, "tls", "ca.key"))
	if err != nil || !bytes.Equal(got, files[filepath.Join(state, "tls", "ca.key")]) {
		t.Fatal("private CA did not restore")
	}
	got, err = os.ReadFile(filepath.Join(restoredState, "metadata", "known-keys", "key.json"))
	if err != nil || !bytes.Contains(got, []byte("vjay")) {
		t.Fatal("local catalog did not restore")
	}
	malicious := opened
	malicious.Files = append(append([]File{}, opened.Files...), File{Root: "config", Path: "../outside", Data: []byte("unsafe")})
	blockedState := filepath.Join(root, "blocked-state")
	if err := Restore(malicious, blockedState, filepath.Join(root, "blocked-config")); err == nil {
		t.Fatal("path traversal accepted")
	}
	if _, err := os.Stat(blockedState); !os.IsNotExist(err) {
		t.Fatal("restore wrote files before validating the full inventory")
	}
	var envelope Envelope
	_ = json.Unmarshal(encoded, &envelope)
	envelope.Ciphertext = envelope.Ciphertext[:len(envelope.Ciphertext)-4] + "AAAA"
	tampered, _ := json.Marshal(envelope)
	if _, err := Open(tampered, "Test-recovery-passphrase-123"); err == nil {
		t.Fatal("tampered capsule accepted")
	}
}
