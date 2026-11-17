package controlplane

import (
	"net/http"
	"testing"
)

func TestWorkspaceFavoritesAPI(t *testing.T) {
	fixture := newAPIFixture(t)
	const path = "/api/v1/auth/workspace-favorites"
	response, raw := fixture.request(http.MethodPut, path, map[string]any{"workspace": "assurance", "starred": true})
	if response.StatusCode != http.StatusOK || string(raw) != "{\"favorites\":[\"assurance\"]}\n" {
		t.Fatalf("save favorite: status %d, %s", response.StatusCode, raw)
	}
	response, raw = fixture.request(http.MethodGet, path, nil)
	if response.StatusCode != http.StatusOK || string(raw) != "{\"favorites\":[\"assurance\"]}\n" {
		t.Fatalf("read favorite: status %d, %s", response.StatusCode, raw)
	}
	response, _ = fixture.request(http.MethodPut, path, map[string]any{"workspace": "assurance", "starred": false, "username": "other-user"})
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("accepted client-supplied account selector: %d", response.StatusCode)
	}
	response, raw = fixture.request(http.MethodPut, path, map[string]any{"workspace": "assurance", "starred": false})
	if response.StatusCode != http.StatusOK || string(raw) != "{\"favorites\":[]}\n" {
		t.Fatalf("remove favorite: status %d, %s", response.StatusCode, raw)
	}
}
