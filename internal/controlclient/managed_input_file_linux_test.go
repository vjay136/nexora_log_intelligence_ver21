package controlclient

import (
	"os"
	"path/filepath"
	"testing"

	"nexora.local/nli-v20/internal/protocol"
)

func TestManagedInputFileSafetyAndLifecycle(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "agent.json")
	if err := os.WriteFile(config, validAgentConfig(t, root, "auto"), 0640); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(root, "input.log")
	retired := filepath.Join(root, "retired.log")
	for _, file := range []string{active, retired} {
		if err := os.WriteFile(file, []byte("log line\n"), 0640); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(retired, filepath.Join(root, "link.log")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"input.log", "link.log", "../retired.log"} {
		if _, err := manageInputFile(root, config, protocol.ManagedInputFileRequest{Action: "delete", Source: name}); err == nil {
			t.Fatalf("delete accepted for %q", name)
		}
	}
	copy := protocol.ManagedInputFileRequest{Action: "copy", Source: "retired.log", Destination: "archive.log"}
	if _, err := manageInputFile(root, config, copy); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "archive.log")
	if data, err := os.ReadFile(archive); err != nil || string(data) != "log line\n" {
		t.Fatalf("unexpected managed copy: %q, %v", data, err)
	}
	if _, err := manageInputFile(root, config, copy); err == nil {
		t.Fatal("existing destination was accepted")
	}
	deleteRequest := protocol.ManagedInputFileRequest{Action: "delete", Source: "retired.log"}
	if _, err := manageInputFile(root, config, deleteRequest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(retired); !os.IsNotExist(err) {
		t.Fatalf("managed delete left the file: %v", err)
	}
	if files, ok := listManagedInputFiles(root); !ok || len(files) != 3 || files[0] != "agent.json" || files[1] != "archive.log" || files[2] != "input.log" {
		t.Fatalf("unexpected disk inventory: %v, %v", files, ok)
	}
	aliasedRoot := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, aliasedRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := manageInputFile(aliasedRoot, config, protocol.ManagedInputFileRequest{Action: "delete", Source: "archive.log"}); err == nil {
		t.Fatal("symlinked managed directory was accepted")
	}
}
