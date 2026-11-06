package familyprofile

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAddOnCannotReplaceProtectedBuiltinIdentity(t *testing.T) {
	builtins := mustBuiltins(t)
	original := findProfile(t, builtins, "nli_reviewed_imap")
	pack := clonePack(t, original.Pack)
	pack.Author = "new-add-on"
	directory := t.TempDir()
	sealAndWrite(t, directory, pack)
	profiles, diagnostics, err := ResolveResilient(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != len(builtins) || len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Reason, "protected built-in") {
		t.Fatalf("conflicting addon not isolated: %+v %+v", profiles, diagnostics)
	}
	preserved := findProfile(t, profiles, original.Pack.ID)
	if preserved.Managed || preserved.Pack.SHA256 != original.Pack.SHA256 {
		t.Fatal("new pack displaced existing built-in")
	}
	if _, err = Resolve(directory); err == nil {
		t.Fatal("strict generation accepted builtin replacement")
	}
	encoded, _ := json.Marshal(pack)
	if _, err = CompileManagedPayload("conflicting", encoded); err == nil {
		t.Fatal("controller memory generation accepted builtin replacement")
	}
}
