package bundlehistory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func testVerify(data []byte) (string, error) {
	var value struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(value.Value))
	return hex.EncodeToString(sum[:]), nil
}
func TestHistoryBoundedCanonicalAndRejectsTampering(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "parser-assets")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		data := []byte(fmt.Sprintf("{\n  \"value\" : \"generation-%d\"\n}", i))
		if err := os.WriteFile(filepath.Join(dir, "bundle.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := Preserve(dir, testVerify); err != nil {
			t.Fatal(err)
		}
	}
	history := filepath.Join(filepath.Dir(dir), "parser-assets-history")
	entries, err := os.ReadDir(history)
	if err != nil || len(entries) != Limit {
		t.Fatalf("retention=%d error=%v", len(entries), err)
	}
	for _, entry := range entries {
		path := filepath.Join(history, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) == 0 || data[0] != '{' || string(data[len(data)-1:]) != "}" {
			t.Fatal("noncanonical archive")
		}
		info, _ := os.Stat(path)
		// Windows exposes a reduced permission model; Unix permission coverage is
		// meaningful on the packaged Linux release gate.
		if info.Mode().Perm()&0077 != 0 && os.PathSeparator == '/' {
			t.Fatal("archive is not private")
		}
	}
	victim := filepath.Join(history, entries[0].Name())
	if err := os.WriteFile(victim, []byte(`{"value":"tampered"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Preserve(dir, testVerify); err == nil {
		t.Fatal("tampered archive accepted")
	}
}
func TestHistoryWriteFailureKeepsPriorGeneration(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "parser-assets")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"value":"important"}`)
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "parser-assets-history"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Preserve(dir, testVerify); err == nil {
		t.Fatal("archive failure hidden")
	}
	actual, _ := os.ReadFile(filepath.Join(dir, "bundle.json"))
	if string(actual) != string(original) {
		t.Fatal("prior generation changed")
	}
}
