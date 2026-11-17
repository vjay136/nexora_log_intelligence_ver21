package auth_test

import (
	"slices"
	"testing"

	"nexora.local/nli-v20/internal/auth"
)

func TestWorkspaceFavoritesPersistPerUser(t *testing.T) {
	path := tmpPath(t)
	store, err := auth.NewStoreWithBootstrap(path, "Test-admin-password-123!")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateUser(auth.RoleAdmin, "viewer", "viewer@example.test", auth.RoleViewer, "Test-viewer-password-123!"); err != nil {
		t.Fatal(err)
	}
	for _, workspace := range []string{"assurance", "outputs", "assurance"} {
		if _, err := store.SetWorkspaceFavorite("admin", auth.RoleAdmin, workspace, true); err != nil {
			t.Fatal(err)
		}
	}
	store, err = auth.NewStoreWithBootstrap(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := store.WorkspaceFavorites("admin", auth.RoleAdmin); !slices.Equal(got, []string{"assurance", "outputs"}) {
		t.Fatalf("unexpected favorites after restart: %v", got)
	}
	if len(store.WorkspaceFavorites("viewer", auth.RoleViewer)) != 0 || len(store.WorkspaceFavorites("admin", auth.RoleViewer)) != 0 {
		t.Fatal("favorites leaked across accounts or roles")
	}
	if got, err := store.SetWorkspaceFavorite("admin", auth.RoleAdmin, "assurance", false); err != nil || !slices.Equal(got, []string{"outputs"}) {
		t.Fatalf("removing favorite: %v (%v)", got, err)
	}
}
