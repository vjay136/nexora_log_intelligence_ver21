package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nexora.local/nli-v20/internal/aiprovider"
	"testing"
)

func TestAIProviderCredentialRemovalLifecyclePurgesHistoryAndPreservesSibling(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	cfg := aiProviderConfig{ID: "removable", Provider: "openai", Model: "test", BaseURL: "https://api.openai.com/v1", Enabled: true, APIKey: "synthetic-test-key-1234", Headers: map[string]string{"X-Vendor-Token": "synthetic-header-key"}}
	for i := 0; i < 2; i++ {
		if err := f.server.storeAIProvider(ctx, cfg, "admin"); err != nil {
			t.Fatal(err)
		}
	}
	sibling := cfg
	sibling.ID = "sibling"
	if err := f.server.storeAIProvider(ctx, sibling, "admin"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/admin/ai-provider", "/api/v1/admin/ai-provider?action=clear-credential"} {
		method := http.MethodDelete
		if len(path) > len("/api/v1/admin/ai-provider") {
			method = http.MethodPost
		}
		response, _ := f.request(method, path, map[string]any{})
		if response.StatusCode != 400 {
			t.Fatal("removal accepted missing explicit ID")
		}
	}
	response, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=clear-credential&id=removable", map[string]any{})
	if response.StatusCode != 200 || decodeMap(t, body)["profile_retained"] != true {
		t.Fatalf("clear failed %d %s", response.StatusCode, body)
	}
	loaded, _, err := f.server.loadAIProviderByID(ctx, "removable")
	if err != nil || loaded.Enabled || !loaded.CredentialRemoved || loaded.APIKey != "" || loaded.LegacyAPIKey != "" || loaded.Credential != (aiprovider.Credential{}) || len(loaded.Headers) != 0 {
		t.Fatalf("clear retained credential material: error %v", err)
	}
	if _, err := f.server.metadata.Get(ctx, "ai-provider", "removable", 2); err == nil {
		t.Fatal("previous credential revision survived purge")
	}
	if _, _, err := f.server.loadAIProviderByID(ctx, "sibling"); err != nil {
		t.Fatal("sibling removed")
	}
	response, _ = f.request(http.MethodPut, "/api/v1/admin/ai-provider", map[string]any{"id": "removable", "provider": "openai", "model": "test", "endpoint": "https://api.openai.com/v1", "enabled": true})
	if response.StatusCode != 400 {
		t.Fatal("reenabled removed credential without fresh key")
	}
	response, body = f.request(http.MethodDelete, "/api/v1/admin/ai-provider?id=removable", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("delete %s", body)
	}
	if _, _, err := f.server.loadAIProviderByID(ctx, "removable"); err == nil {
		t.Fatal("deleted profile survived")
	}
	if _, _, err := f.server.loadAIProviderByID(ctx, "sibling"); err != nil {
		t.Fatal("delete touched sibling")
	}
}

func TestAIProviderLateProbeCannotResurrectRemovedCredentials(t *testing.T) {
	for _, retain := range []bool{true, false} {
		t.Run(map[bool]string{true: "clear", false: "delete"}[retain], func(t *testing.T) {
			t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
			f := newAPIFixture(t)
			entered := make(chan struct{})
			release := make(chan struct{})
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				close(entered)
				<-release
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "OK"}, "finish_reason": "stop"}}})
			}))
			defer provider.Close()
			cfg := aiProviderConfig{ID: "delayed", Provider: "openai_compatible", Model: "test", BaseURL: provider.URL, Enabled: true, APIKey: "synthetic-test-key-1234"}
			if err := f.server.storeAIProvider(context.Background(), cfg, "admin"); err != nil {
				t.Fatal(err)
			}
			done := make(chan []byte, 1)
			go func() {
				_, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider/test", map[string]any{"id": "delayed"})
				done <- body
			}()
			<-entered
			if _, err := f.server.removeAIProviderCredential(context.Background(), "delayed", "admin", retain); err != nil {
				t.Fatal(err)
			}
			close(release)
			result := decodeMap(t, <-done)
			if result["connected"] != false || result["status"] != "configuration_changed" {
				t.Fatal("late probe falsely marked old profile ready")
			}
			loaded, sealed, err := f.server.loadAIProviderByID(context.Background(), "delayed")
			if retain {
				if err != nil || loaded.Enabled || !loaded.CredentialRemoved || sealed.LastTestAt != nil {
					t.Fatal("late probe resurrected cleared profile")
				}
			} else if err == nil {
				t.Fatal("late probe recreated deleted profile")
			}
		})
	}
}

func TestAIProviderLegacyExactCaseRemovalAndCanonicalNewIDs(t *testing.T) {
	f := newAPIFixture(t)
	ctx := context.Background()
	t.Setenv("TEST_AI_CASE_KEY", "synthetic-case-key-1234")
	cfg := aiProviderConfig{ID: "UPPER", Provider: "openai", Model: "test", BaseURL: "https://api.openai.com/v1", Enabled: true, Credential: aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: "TEST_AI_CASE_KEY"}}
	if err := f.server.storeAIProvider(ctx, cfg, "admin"); err != nil {
		t.Fatal(err)
	}
	cfg.ID = "upper"
	if err := f.server.storeAIProvider(ctx, cfg, "admin"); err != nil {
		t.Fatal(err)
	}
	response, _ := f.request(http.MethodPut, "/api/v1/admin/ai-provider", map[string]any{"id": "NEWUPPER", "provider": "openai", "model": "test", "endpoint": "https://api.openai.com/v1", "credential_source": "env_ref", "credential_ref": "TEST_AI_CASE_KEY"})
	if response.StatusCode != 400 {
		t.Fatal("new noncanonical ID accepted")
	}
	response, body := f.request(http.MethodDelete, "/api/v1/admin/ai-provider?id=UPPER", map[string]any{})
	if response.StatusCode != 200 {
		t.Fatalf("exact legacy deletion failed %s", body)
	}
	if _, _, err := f.server.loadAIProviderByID(ctx, "UPPER"); err == nil {
		t.Fatal("uppercase exact row retained")
	}
	if _, _, err := f.server.loadAIProviderByID(ctx, "upper"); err != nil {
		t.Fatal("deletion silently normalized into lowercase sibling")
	}
}
