//go:build linux

package recovery

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
)

func TestRestoreOwnedFilesReadableByServiceIdentity(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Linux root is needed to verify restored service ownership")
	}
	account, err := user.Lookup("nobody")
	if err != nil {
		t.Skip("service test identity is unavailable")
	}
	runuser, err := exec.LookPath("runuser")
	if err != nil {
		t.Skip("runuser is unavailable")
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		t.Fatal(err)
	}
	gid, err := strconv.Atoi(account.Gid)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chmod(filepath.Dir(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	state, config := filepath.Join(root, "state"), filepath.Join(root, "config")
	contents := Contents{Schema: 1, Files: []File{{Root: "state", Path: "users.json", Mode: 0600, Data: []byte("service-identity")}, {Root: "state", Path: "tls/server.key", Mode: 0640, Data: []byte("service-tls-key")}, {Root: "state", Path: "tls/ca.key", Mode: 0600, Data: []byte("root-ca-key")}, {Root: "config", Path: "controller.env", Mode: 0640, Data: []byte("service-config")}}}
	if err := RestoreOwned(contents, state, config, uid, gid); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(state, "users.json"), filepath.Join(state, "tls/server.key"), filepath.Join(config, "controller.env")} {
		if output, err := exec.Command(runuser, "-u", account.Username, "--", "cat", path).CombinedOutput(); err != nil {
			t.Fatalf("service cannot read restored %s: %v %s", path, err, output)
		}
	}
	if err := exec.Command(runuser, "-u", account.Username, "--", "cat", filepath.Join(state, "tls/ca.key")).Run(); err == nil {
		t.Fatal("private CA key became readable by the service")
	}
}
