package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"nexora.local/nli-v20/internal/auditlog"
	metastore "nexora.local/nli-v20/internal/store"
)

func TestAIProviderCatalogRecoversUnreadableProfileWithoutLosingOtherProviders(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	_, err := f.server.metadata.Put(ctx, metastore.Document{
		Collection: "ai-provider", ID: "unreadable", State: "configured",
		Payload: json.RawMessage(`{"nonce":"invalid-nonce","ciphertext":"opaque-test-value"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodGet, "/api/v1/admin/ai-provider?id=unreadable", nil)
	view := decodeMap(t, body)
	provider := view["provider"].(map[string]any)
	capabilities := view["capabilities"].(map[string]any)
	if response.StatusCode != http.StatusOK || view["store_status"] != "degraded" ||
		view["blocked_profile_count"] != float64(1) || provider["id"] != "unreadable" ||
		provider["status"] != "unreadable" || capabilities["available"] != false {
		t.Fatalf("unreadable catalog did not expose its recovery state: %d %s", response.StatusCode, body)
	}
	if bytes.Contains(body, []byte("opaque-test-value")) || bytes.Contains(body, []byte("invalid-nonce")) {
		t.Fatal("catalog leaked encryption material")
	}
	if err := f.server.storeAIProvider(ctx, aiProviderConfig{
		ID: "healthy", Provider: "openai", Model: "test-model", Enabled: true, APIKey: "synthetic-test-credential",
	}, "admin"); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := f.server.loadAIProvider(ctx)
	if err != nil || !cfg.Enabled || cfg.ID != "healthy" {
		t.Fatalf("readable provider was not available for routing: %v", err)
	}
	response, body = f.request(http.MethodPut, "/api/v1/admin/ai-provider", map[string]any{
		"id": "unreadable", "provider": "openai", "model": "test-model", "enabled": true,
		"api_key": "replacement-test-credential",
	})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["saved"] != true {
		t.Fatalf("could not save replacement provider credential: %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodGet, "/api/v1/admin/ai-provider?id=unreadable", nil)
	view = decodeMap(t, body)
	if response.StatusCode != http.StatusOK || view["store_status"] != "available" ||
		view["blocked_profile_count"] != float64(0) || view["provider"].(map[string]any)["configured"] != true {
		t.Fatalf("replacement credential did not recover profile: %d %s", response.StatusCode, body)
	}
}

type failingAIProviderListStore struct {
	metastore.Store
}

func (s failingAIProviderListStore) Get(ctx context.Context, collection, id string, revision int) (metastore.Document, error) {
	if collection == aiProviderRoutingCollection {
		return metastore.Document{}, errors.New("test internal routing store detail")
	}
	return s.Store.Get(ctx, collection, id, revision)
}

func (s failingAIProviderListStore) List(ctx context.Context, collection string, query metastore.Query) ([]metastore.Document, error) {
	if collection == "ai-provider" {
		return nil, errors.New("test internal provider store detail")
	}
	return s.Store.List(ctx, collection, query)
}

func TestAIProviderCatalogSurvivesMetadataListFailure(t *testing.T) {
	s := Server{assurance: &assuranceVault{}, metadata: failingAIProviderListStore{}}
	view := s.aiProviderCatalog(context.Background(), "")
	if view["store_status"] != "unavailable" || view["profiles_unavailable"] != true ||
		view["capabilities"].(map[string]any)["available"] != false || view["templates"] == nil {
		t.Fatalf("metadata failure removed the recovery catalog: %#v", view)
	}
	if view["store_message"] == "test internal provider store detail" {
		t.Fatal("catalog exposed internal error detail")
	}
}

func TestAIProviderCatalogHistoricalReadFailureStaysOutOfNotifications(t *testing.T) {
	event := auditlog.Event{
		Action: http.MethodGet, Resource: "/api/v1/admin/ai-provider", Outcome: "failure",
		Details: map[string]string{"detail": "status=503 duration_ms=10"},
	}
	if _, _, actionable := classifyNotificationEvent(event); actionable {
		t.Fatal("historical provider catalog probe generated an actionable notification")
	}
	event.Action = http.MethodPut
	if _, _, actionable := classifyNotificationEvent(event); !actionable {
		t.Fatal("provider save failure lost its actionable notification")
	}
}
