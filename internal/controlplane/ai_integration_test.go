package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"nexora.local/nli-v20/internal/aiprovider"
	metastore "nexora.local/nli-v20/internal/store"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAIConfiguredFallbackMasksCompleteContextAndStaysReviewOnly(t *testing.T) {
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	f := newAPIFixture(t)
	var first, second, other atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { first.Add(1); w.WriteHeader(503) }))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		second.Add(1)
		var input map[string]any
		_ = json.NewDecoder(r.Body).Decode(&input)
		data, _ := json.Marshal(input)
		for _, private := range []string{"private@example.test", "wanted-private-password", "raw-private-password"} {
			if bytes.Contains(data, []byte(private)) {
				t.Error("provider prompt retained private context")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"name":"review","parser":{"pattern_type":"regex","pattern":"^(?P<message>.*)$","fields":[{"name":"message","source":"group:message","type":"string"}]}}`}}}})
	}))
	defer fallback.Close()
	unrelated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { other.Add(1); w.WriteHeader(500) }))
	defer unrelated.Close()
	for _, cfg := range []aiProviderConfig{{ID: "primary", Provider: "nvidia_nim", Model: "deepseek-ai/deepseek-v4.1-flash", BaseURL: primary.URL, Enabled: true, Priority: 0, FallbackGroup: "review", APIKey: "fake-provider-key-1234"}, {ID: "fallback", Provider: "nvidia_nim", Model: "minimaxai/minimax-m3.1-flash", BaseURL: fallback.URL, Enabled: true, Priority: 1, FallbackGroup: "review", APIKey: "fake-provider-key-1234"}, {ID: "unrelated", Provider: "openai_compatible", Model: "test", BaseURL: unrelated.URL, Enabled: true, Priority: 2, FallbackGroup: "other", APIKey: "fake-provider-key-1234"}} {
		if err := f.server.storeAIProvider(context.Background(), cfg, "admin"); err != nil {
			t.Fatal(err)
		}
	}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/ai-proposal", map[string]any{"raw": "owner=private@example.test status=FAILED password=raw-private-password", "service": "custom", "desired": map[string]any{"email": "private@example.test", "password": "wanted-private-password", "status": "DELIVERED"}})
	if response.StatusCode != 200 {
		t.Fatalf("proposal status %d: %s", response.StatusCode, body)
	}
	result := decodeMap(t, body)
	if result["provider_id"] != "fallback" || result["provider_attempts"] != float64(2) || result["published"] != false || result["requires_human_approval"] != true {
		t.Fatalf("wrong routed review metadata: %s", body)
	}
	if first.Load() != 1 || second.Load() != 1 || other.Load() != 0 {
		t.Fatal("fallback crossed group or duplicated provider requests")
	}
	_, catalog := f.request(http.MethodGet, "/api/v1/admin/ai-provider", nil)
	if bytes.Contains(catalog, []byte("fake-provider-key")) || bytes.Contains(catalog, []byte("ciphertext")) {
		t.Fatal("provider catalog leaked secret")
	}
}
func TestAICapabilitiesMissingCredentialAndCorruptProfileIsolation(t *testing.T) {
	f := newAPIFixture(t)
	t.Setenv("NLI_AI_TEST_EMPTY", "")
	if err := f.server.storeAIProvider(context.Background(), aiProviderConfig{ID: "missing", Provider: "openai", Model: "test", BaseURL: "https://api.openai.com/v1", Enabled: true, Credential: aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: "NLI_AI_TEST_EMPTY"}}, "admin"); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodGet, "/api/v1/ai/capabilities", nil)
	if response.StatusCode != 200 || decodeMap(t, body)["available"] != false {
		t.Fatalf("missing env falsely ready: %s", body)
	}
	if err := f.server.storeAIProvider(context.Background(), aiProviderConfig{ID: "healthy", Provider: "openai", Model: "test", BaseURL: "https://api.openai.com/v1", Enabled: true, APIKey: "fake-key-value-1234"}, "admin"); err != nil {
		t.Fatal(err)
	}
	_, err := f.server.metadata.Put(context.Background(), metastore.Document{Collection: "ai-provider", ID: "broken", State: "configured", Payload: []byte(`{"nonce":"invalid","ciphertext":"invalid"}`)})
	if err != nil {
		t.Fatal(err)
	}
	_, body = f.request(http.MethodGet, "/api/v1/ai/capabilities", nil)
	if decodeMap(t, body)["available"] != true {
		t.Fatalf("bad sibling blocked healthy provider: %s", body)
	}
	response, body = f.request(http.MethodGet, "/api/v1/admin/ai-provider?id=broken", nil)
	if response.StatusCode != 503 || strings.Contains(string(body), "ciphertext") {
		t.Fatal("explicit corrupt profile must fail safely")
	}
}
func TestAIProviderDisabledCannotGenerateAndOldSaveKeepsExplicitDisabledState(t *testing.T) {
	f := newAPIFixture(t)
	cfg := aiProviderConfig{ID: "disabled", Provider: "openai", Model: "test", BaseURL: "https://api.openai.com/v1", Enabled: false, APIKey: "fake-key-value-1234"}
	if err := f.server.storeAIProvider(context.Background(), cfg, "admin"); err != nil {
		t.Fatal(err)
	}
	response, _ := f.request(http.MethodPost, "/api/v1/assurance/ai-proposal", map[string]any{"raw": "unmatched source", "service": "custom", "desired": map[string]any{"required": "x"}})
	if response.StatusCode != 503 {
		t.Fatal("disabled provider accepted generation")
	}
	response, body := f.request(http.MethodPut, "/api/v1/admin/ai-provider", map[string]any{"id": "disabled", "provider": "openai", "model": "updated", "endpoint": "https://api.openai.com/v1"})
	if response.StatusCode != 200 {
		t.Fatalf("old save: %d %s", response.StatusCode, body)
	}
	loaded, _, err := f.server.loadAIProviderByID(context.Background(), "disabled")
	if err != nil || loaded.Enabled {
		t.Fatal("omitted enabled unexpectedly enabled disabled profile")
	}
}

func TestAIProviderProbeBindsUnsavedModelAndOnlyVerifiesExactSavedConfiguration(t *testing.T) {
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	f := newAPIFixture(t)
	models := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		models <- request["model"].(string)
		if r.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"OK\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	cfg := aiProviderConfig{ID: "nvidia", Name: "NVIDIA", Provider: "nvidia_nim", Model: "saved-model", BaseURL: server.URL + "/v1/chat/completions", Enabled: true, APIKey: "fake-probe-key-1234"}
	if err := f.server.storeAIProvider(context.Background(), cfg, "admin"); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider/test", map[string]any{"id": "nvidia", "model": "minimaxai/minimax-m3.1-flash"})
	result := decodeMap(t, body)
	if response.StatusCode != 200 || result["connected"] != true {
		t.Fatalf("draft probe failed %d %s", response.StatusCode, body)
	}
	info := result["tested_config"].(map[string]any)
	if info["model"] != "minimaxai/minimax-m3.1-flash" || info["configuration"] != "unsaved_draft" || len(info["config_checksum"].(string)) != 64 {
		t.Fatal("test acknowledgement did not identify exact draft")
	}
	if <-models != "minimaxai/minimax-m3.1-flash" {
		t.Fatal("stored model tested instead of displayed model")
	}
	_, sealed, err := f.server.loadAIProviderByID(context.Background(), "nvidia")
	if err != nil || sealed.LastTestAt != nil {
		t.Fatal("unsaved test incorrectly verified stored model")
	}
	response, body = f.request(http.MethodPost, "/api/v1/admin/ai-provider/test", map[string]any{"id": "nvidia", "model": "saved-model", "stream": true, "image_url": "https://assets.ngc.nvidia.com/products/example.jpg"})
	result = decodeMap(t, body)
	if response.StatusCode != 200 || result["connected"] != true || result["tested_config"].(map[string]any)["test_mode"] != "sse" {
		t.Fatalf("SSE probe failed %d %s", response.StatusCode, body)
	}
	<-models
	_, sealed, err = f.server.loadAIProviderByID(context.Background(), "nvidia")
	if err != nil || sealed.LastTestAt == nil {
		t.Fatal("exact saved model success was not recorded")
	}
}
