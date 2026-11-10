package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestAIProviderDefaultIsExclusiveAndExplicitProbeIsIndependent(t *testing.T) {
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	f := newAPIFixture(t)
	ctx := context.Background()
	var defaultCalls, alternateCalls atomic.Int32
	pinned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultCalls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer pinned.Close()
	alternate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		alternateCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "OK"}, "finish_reason": "stop"}},
		})
	}))
	defer alternate.Close()
	defaultCfg := aiProviderConfig{ID: "exclusive", Provider: "openai_compatible", Model: "pinned-model", BaseURL: pinned.URL, Enabled: true, Priority: 10, FallbackGroup: "review", APIKey: "fake-provider-key-1234"}
	alternateCfg := aiProviderConfig{ID: "alternate", Provider: "openai_compatible", Model: "alternate-model", BaseURL: alternate.URL, Enabled: true, Priority: 1, FallbackGroup: "review", APIKey: "fake-provider-key-5678"}
	for _, cfg := range []aiProviderConfig{defaultCfg, alternateCfg} {
		if err := f.server.storeAIProvider(ctx, cfg, "admin"); err != nil {
			t.Fatal(err)
		}
	}
	response, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=set-default&id=exclusive", map[string]any{})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["default_profile_id"] != "exclusive" {
		t.Fatalf("default was not saved: %d %s", response.StatusCode, body)
	}
	_, body = f.request(http.MethodGet, "/api/v1/admin/ai-provider", nil)
	catalog := decodeMap(t, body)
	if catalog["default_profile_id"] != "exclusive" || catalog["provider"].(map[string]any)["model"] != "pinned-model" {
		t.Fatalf("catalog did not report persisted selection: %s", body)
	}
	response, _ = f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{"mode": "ask", "workspace": "field-mapping", "question": "What can I map?"})
	if response.StatusCode < 400 || defaultCalls.Load() != 1 || alternateCalls.Load() != 0 {
		t.Fatal("workspace AI fell back from the selected provider")
	}
	cfg, _, err := f.server.loadAIProvider(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, attempts, err := f.server.callConfiguredAI(ctx, cfg, "synthetic test"); err == nil || attempts != 1 || alternateCalls.Load() != 0 {
		t.Fatal("Assurance AI fell back from the selected provider")
	}
	response, body = f.request(http.MethodPost, "/api/v1/admin/ai-provider/test", map[string]any{"id": "alternate"})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["connected"] != true || alternateCalls.Load() != 1 {
		t.Fatalf("explicit test did not reach the selected alternate: %d %s", response.StatusCode, body)
	}
	defaultCfg.Enabled = false
	if err := f.server.storeAIProvider(ctx, defaultCfg, "admin"); err != nil {
		t.Fatal(err)
	}
	_, body = f.request(http.MethodGet, "/api/v1/ai/capabilities", nil)
	capabilities := decodeMap(t, body)
	if capabilities["default_profile_id"] != "exclusive" || capabilities["status"] != "default_unavailable" || capabilities["available"] != false {
		t.Fatalf("disabled default left AI available: %s", body)
	}
	if _, _, err := f.server.loadAIProvider(ctx); err == nil {
		t.Fatal("disabled default fell back to the alternate")
	}
	response, _ = f.request(http.MethodDelete, "/api/v1/admin/ai-provider?id=exclusive", map[string]any{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("could not delete default profile: %d", response.StatusCode)
	}
	if _, _, err := f.server.loadAIProvider(ctx); err == nil {
		t.Fatal("deleted default fell back to the alternate")
	}
	response, body = f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=clear-default", map[string]any{})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["default_profile_id"] != "" {
		t.Fatalf("automatic routing was not restored: %d %s", response.StatusCode, body)
	}
	cfg, _, err = f.server.loadAIProvider(ctx)
	if err != nil || cfg.ID != "alternate" || cfg.Exclusive {
		t.Fatalf("automatic routing did not select the remaining provider: %v", err)
	}
}
