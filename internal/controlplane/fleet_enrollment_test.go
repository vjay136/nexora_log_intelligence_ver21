package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/protocol"
)

func TestFleetEnrollmentGrantsBindAndRecoverOneNode(t *testing.T) {
	f := newAPIFixture(t)
	dir := filepath.Join(f.state, "enrollment", "grants")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	grant := func(digit, node string, lifetime time.Duration) string {
		t.Helper()
		token := fleetGrantPrefix + strings.Repeat(digit, 64)
		deadline := time.Now().UTC().Add(lifetime).Format(time.RFC3339)
		if err := os.WriteFile(filepath.Join(dir, enrollmentDigest(token)), []byte(deadline+" "+node+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		return token
	}
	register := func(id, token, proof string, want int) protocol.RegisterResponse {
		t.Helper()
		data, _ := json.Marshal(protocol.RegisterRequest{ID: id, Hostname: "test-host", Arch: "x86_64"})
		req, _ := http.NewRequest(http.MethodPost, f.http.URL+"/api/v1/node/register", bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-NLI-Enrollment", token)
		req.Header.Set("X-NLI-Enrollment-Proof", proof)
		resp, err := f.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("registration status = %d, want %d", resp.StatusCode, want)
		}
		var result protocol.RegisterResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	node := fleetNodePrefix + strings.Repeat("a", 64)
	other := fleetNodePrefix + strings.Repeat("b", 64)
	proof := strings.Repeat("c", 64)
	key := grant("1", "", time.Hour)
	register(node, "", proof, http.StatusUnauthorized)
	register(node, key, "", http.StatusUnauthorized)
	first := register(node, key, proof, http.StatusCreated)
	register(other, key, proof, http.StatusUnauthorized)
	register(node, key, strings.Repeat("d", 64), http.StatusUnauthorized)
	register(node, grant("2", "", time.Hour), proof, http.StatusUnauthorized)
	register(other, grant("3", "", -time.Hour), proof, http.StatusUnauthorized)
	retry := register(node, key, proof, http.StatusCreated)
	replacement := grant("4", node, time.Hour)
	register(other, replacement, proof, http.StatusUnauthorized)
	last := register(node, replacement, proof, http.StatusCreated)
	register(other, key, proof, http.StatusUnauthorized)
	register(node, key, proof, http.StatusUnauthorized)
	persisted := fleet.New(filepath.Join(f.state, "fleet.json"))
	if len(persisted.All()) != 1 || !persisted.Authorize(node, last.NodeToken) || persisted.Authorize(node, first.NodeToken) || persisted.Authorize(node, retry.NodeToken) {
		t.Fatal("expected exactly one durable node with only its replacement credential active")
	}
}
