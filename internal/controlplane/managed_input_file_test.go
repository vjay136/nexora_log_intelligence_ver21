package controlplane

import (
	"net/http"
	"testing"
	"time"

	"nexora.local/nli-v20/internal/fleet"
	"nexora.local/nli-v20/internal/protocol"
)

func TestManagedInputOperationRequiresExactApprovalAndQueuesOfflineClient(t *testing.T) {
	f := newAPIFixture(t)
	f.server.fleet.Register(&fleet.Node{ID: "managed-node", Hostname: "managed", Version: "22.1.0"})
	f.server.fleet.MarkStale(0)
	f.refreshCSRF()
	response, body := f.request(http.MethodPost, "/api/v1/users", map[string]any{
		"username": "managed-reviewer", "email": "managed-reviewer@example.test",
		"role": "reviewer", "password": "Managed-reviewer-password-12345!",
	})
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create reviewer %d: %s", response.StatusCode, body)
	}
	payload := map[string]any{"action": "delete", "source": "../secrets"}
	proposal := map[string]any{
		"name": "Delete retired managed input file",
		"type": protocol.ManagedInputFileCommand, "target_nodes": []string{"managed-node"},
		"payload": payload,
	}
	response, body = f.request(http.MethodPost, "/api/v1/rollouts", proposal)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("traversal proposal accepted %d: %s", response.StatusCode, body)
	}
	payload["source"] = "retired.log"
	response, body = f.request(http.MethodPost, "/api/v1/rollouts", proposal)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("valid proposal rejected %d: %s", response.StatusCode, body)
	}
	id := decodeMap(t, body)["id"].(string)
	if command, ok := f.server.rollouts.NextForNode("managed-node", time.Minute); ok {
		t.Fatalf("command was delivered before approval: %#v", command)
	}
	f.login("managed-reviewer", "Managed-reviewer-password-12345!")
	f.refreshCSRF()
	response, body = f.request(http.MethodPost, "/api/v1/rollouts/"+id+"/approve", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("approval did not queue offline operation %d: %s", response.StatusCode, body)
	}
	command, ok := f.server.rollouts.NextForNode("managed-node", time.Minute)
	if !ok || command.Type != protocol.ManagedInputFileCommand || string(command.Payload) != `{"action":"delete","source":"retired.log"}` {
		t.Fatalf("unexpected approved command: %#v", command)
	}
}
