//go:build !windows

package output

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPrivilegedWriterPreservesAgentOwnership(t *testing.T) {
	if os.Getenv("NLI_OUTPUT_UID_HELPER") == "1" {
		w, err := NewFileWriter(Config{Type: "file", Path: os.Getenv("NLI_OUTPUT_UID_TARGET"), DeliveryStatePath: os.Getenv("NLI_OUTPUT_UID_WAL")})
		if err != nil {
			t.Fatal(err)
		}
		id := deliveryTestID("unprivileged-agent")
		if err = w.WriteIdentified([]byte(`{"actor":"agent"}`), "normal", id); err != nil {
			t.Fatal(err)
		}
		if err = w.AcknowledgeDeliveries([]string{id}); err != nil {
			t.Fatal(err)
		}
		if err = w.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Skip("requires root to exercise UID transition")
	}
	root := t.TempDir()
	if err := os.Chmod(filepath.Dir(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(root, 1001, 1002); err != nil {
		t.Skipf("container lacks CHOWN capability: %v", err)
	}
	state := filepath.Join(root, "state")
	if err := os.Mkdir(state, 0700); err != nil {
		t.Fatal(err)
	}
	_ = os.Chown(state, 1001, 1002)
	target := filepath.Join(root, "events.jsonl")
	wal := DeliveryJournalPath(state, target, "")
	w, _ := NewFileWriter(Config{Type: "file", Path: target, DeliveryStatePath: wal})
	id := deliveryTestID("privileged-replay")
	if err := w.WriteIdentified([]byte(`{"actor":"replay"}`), "normal", id); err != nil {
		t.Fatal(err)
	}
	if err := w.AcknowledgeDeliveries([]string{id}); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	for _, path := range []string{target, wal, filepath.Dir(wal)} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		owner := info.Sys().(*syscall.Stat_t)
		if owner.Uid != 1001 || owner.Gid != 1002 {
			t.Fatalf("privileged creation/replacement changed agent ownership of %s", filepath.Base(path))
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.Open(executable)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "output-test-helper")
	out, err := os.OpenFile(binary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(out, input)
	_ = input.Close()
	_ = out.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	command := exec.Command(binary, "-test.run=^TestPrivilegedWriterPreservesAgentOwnership$")
	command.Env = append(os.Environ(), "NLI_OUTPUT_UID_HELPER=1", "NLI_OUTPUT_UID_TARGET="+target, "NLI_OUTPUT_UID_WAL="+wal)
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 1001, Gid: 1002}}
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("agent UID could not continue after privileged replay: %v %s", err, result)
	}
}
