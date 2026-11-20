package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"nexora.local/nli-v20/internal/auditlog"
	metastore "nexora.local/nli-v20/internal/store"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPersistedMetadataFailureRetriesAreGroupedWithoutLosingAudit(t *testing.T) {
	f := newAPIFixture(t)
	for i := 0; i < 30; i++ {
		path := []string{"/api/v1/artifacts", "/api/v1/config/templates", "/api/v1/config/kv"}[i%3]
		if _, err := f.server.audit.Append(auditlog.Event{Actor: "admin", Action: "GET", Resource: path, Outcome: "failure", Details: map[string]string{"detail": "status=503 backend_error=metadata_authentication_failed"}}); err != nil {
			t.Fatal(err)
		}
	}
	response, body := f.request(http.MethodGet, "/api/v1/notifications?limit=100", nil)
	if response.StatusCode != 200 {
		t.Fatal(string(body))
	}
	items := decodeMap(t, body)["notifications"].([]any)
	count := 0
	for _, raw := range items {
		if raw.(map[string]any)["title"] == "Controller metadata authentication failed" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("retry storm became %d notifications", count)
	}
	if err := f.server.audit.Verify(); err != nil {
		t.Fatalf("audit integrity: %v", err)
	}
}

type rejectedControllerStore struct {
	metastore.Store
	denied atomic.Bool
}

func (s *rejectedControllerStore) List(ctx context.Context, collection string, query metastore.Query) ([]metastore.Document, error) {
	if !s.denied.Load() {
		return s.Store.List(ctx, collection, query)
	}
	return nil, &metastore.RPCError{Function: "nli_list_documents", Status: 401, Code: "42501", Message: "NLI controller authentication failed"}
}
func (s *rejectedControllerStore) Put(ctx context.Context, doc metastore.Document) (metastore.Document, error) {
	if !s.denied.Load() {
		return s.Store.Put(ctx, doc)
	}
	return metastore.Document{}, &metastore.RPCError{Function: "nli_put_document", Status: 401, Code: "42501", Message: "NLI controller authentication failed"}
}

func TestReportedCatalogAPIsDistinguishBackendAuthenticationAndRecover(t *testing.T) {
	f := newAPIFixture(t)
	store := &rejectedControllerStore{Store: f.server.metadata}
	f.server.metadata = store
	paths := []string{"/api/v1/artifacts", "/api/v1/assurance/parsers", "/api/v1/config/kv", "/api/v1/config/templates", "/api/v1/rollouts", "/api/v1/assurance/cases", "/api/v1/parser-sync", "/api/v1/parser-packs", "/api/v1/fleet/sync", "/api/v1/assurance/runs", "/api/v1/assurance/plans", "/api/v1/assurance/history", "/api/v1/assurance/ai-proposals"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			store.denied.Store(true)
			response, body := f.request(http.MethodGet, path, nil)
			value := decodeMap(t, body)
			if response.StatusCode != http.StatusServiceUnavailable || value["backend_error"] != "metadata_authentication_failed" || value["action"] == nil {
				t.Fatalf("backend failure mistaken for an empty catalog or login failure: %d %s", response.StatusCode, body)
			}
			store.denied.Store(false)
			response, body = f.request(http.MethodGet, path, nil)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("same login did not recover: %d %s", response.StatusCode, body)
			}
		})
	}
}

func TestNotificationActionsRecoverAfterControllerAuthenticationRepair(t *testing.T) {
	f := newAPIFixture(t)
	store := &rejectedControllerStore{Store: f.server.metadata}
	f.server.metadata = store
	response, body := f.request(http.MethodGet, "/api/v1/notifications", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatal(string(body))
	}
	items := decodeMap(t, body)["notifications"].([]any)
	if len(items) == 0 {
		t.Fatal("login audit notification missing")
	}
	id := items[0].(map[string]any)["id"].(string)
	path := "/api/v1/notifications/" + id
	store.denied.Store(true)
	response, body = f.request(http.MethodPost, path, map[string]any{"action": "read"})
	if response.StatusCode != 503 || decodeMap(t, body)["backend_error"] != "metadata_authentication_failed" {
		t.Fatalf("%d %s", response.StatusCode, body)
	}
	store.denied.Store(false)
	for _, action := range []string{"read", "unread", "snooze", "dismissed"} {
		response, body = f.request(http.MethodPost, path, map[string]any{"action": action, "snooze_minutes": 5})
		if response.StatusCode != 200 {
			t.Fatalf("%s: %d %s", action, response.StatusCode, body)
		}
	}
}

func TestMetadataFailureDiagnosticsNeverEchoBackendCredentials(t *testing.T) {
	err := &metastore.RPCError{Status: 401, Code: "42501", Message: "private-secret-from-upstream"}
	data, _ := json.Marshal(metadataFailureDetails(err))
	if strings.Contains(string(data), "private-secret") {
		t.Fatal("backend credentials exposed")
	}
	if got := storageDiagnostic(err.Error()); got != "controller_secret_mismatch_or_rpc_permission" {
		t.Fatalf("wrong remediation: %s", got)
	}
}
