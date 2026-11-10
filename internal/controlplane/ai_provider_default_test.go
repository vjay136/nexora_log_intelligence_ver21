package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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
	response, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=set-order", map[string]any{"profile_order": []string{"exclusive", "alternate"}})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("provider order was not saved: %d %s", response.StatusCode, body)
	}
	cfg, _, err := f.server.loadAIProvider(ctx)
	if err != nil || cfg.ID != "exclusive" || cfg.Exclusive {
		t.Fatalf("saved order did not override numeric priority: %#v %v", cfg.ID, err)
	}
	if _, used, attempts, err := f.server.callConfiguredAI(ctx, cfg, "synthetic test"); err != nil || used != "alternate" || attempts != 2 {
		t.Fatalf("automatic fallback did not try providers in saved order: %s %d %v", used, attempts, err)
	}
	response, body = f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{"mode": "ask", "workspace": "field-mapping", "question": "What can I map?"})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["provider_routing"] != "standby" {
		t.Fatalf("successful standby use was not reported: %d %s", response.StatusCode, body)
	}
	defaultCalls.Store(0)
	alternateCalls.Store(0)
	response, body = f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=set-default&id=exclusive", map[string]any{})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["default_profile_id"] != "exclusive" {
		t.Fatalf("default was not saved: %d %s", response.StatusCode, body)
	}
	response, body = f.request(http.MethodPost, "/api/v1/admin/ai-provider?action=set-order", map[string]any{"profile_order": []string{"alternate", "exclusive"}})
	if response.StatusCode != http.StatusOK || decodeMap(t, body)["default_profile_id"] != "exclusive" {
		t.Fatalf("order update dropped default: %d %s", response.StatusCode, body)
	}
	_, body = f.request(http.MethodGet, "/api/v1/admin/ai-provider", nil)
	catalog := decodeMap(t, body)
	order := catalog["profile_order"].([]any)
	if catalog["default_profile_id"] != "exclusive" || catalog["provider"].(map[string]any)["model"] != "pinned-model" ||
		len(order) != 2 || order[0] != "alternate" || order[1] != "exclusive" {
		t.Fatalf("catalog did not report persisted selection: %s", body)
	}
	response, _ = f.request(http.MethodPost, "/api/v1/ai/workspace", map[string]any{"mode": "ask", "workspace": "field-mapping", "question": "What can I map?"})
	if response.StatusCode < 400 || defaultCalls.Load() != 2 || alternateCalls.Load() != 0 {
		t.Fatal("workspace AI fell back from the selected provider")
	}
	cfg, _, err = f.server.loadAIProvider(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, attempts, err := f.server.callConfiguredAI(ctx, cfg, "synthetic test"); err == nil || attempts != 2 || alternateCalls.Load() != 0 {
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
	restored := decodeMap(t, body)
	if response.StatusCode != http.StatusOK || restored["default_profile_id"] != "" || len(restored["profile_order"].([]any)) != 2 {
		t.Fatalf("automatic routing was not restored: %d %s", response.StatusCode, body)
	}
	cfg, _, err = f.server.loadAIProvider(ctx)
	if err != nil || cfg.ID != "alternate" || cfg.Exclusive {
		t.Fatalf("automatic routing did not select the remaining provider: %v", err)
	}
}

func TestAIProviderRoutingReachesFourthProfileWithinDeadline(t *testing.T) {
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	f := newAPIFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var unavailableCalls, standbyCalls atomic.Int32
	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unavailableCalls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unavailable.Close()
	standby := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		standbyCalls.Add(1)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer standby.Close()
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("profile-%d", i)
		base := unavailable.URL
		if i == 4 {
			base = standby.URL
		}
		if err := f.server.storeAIProvider(ctx, aiProviderConfig{ID: id, Name: id, Provider: "openai_compatible", Model: "test-model", BaseURL: base, Enabled: true, Priority: i, APIKey: "synthetic-provider-key-1234", FallbackGroup: "review"}, "admin"); err != nil {
			t.Fatal(err)
		}
	}
	cfg, _, err := f.server.loadAIProvider(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, provider, attempts, err := f.server.callConfiguredAI(ctx, cfg, "synthetic test")
	if err != nil || provider != "profile-4" || attempts != 4 || unavailableCalls.Load() != 3 || standbyCalls.Load() != 1 {
		t.Fatalf("ordered routing did not reach the fourth profile: provider=%q attempts=%d err=%v", provider, attempts, err)
	}
}
