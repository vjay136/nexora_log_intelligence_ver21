package controlplane

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIProviderDiscoveryUsesSavedCredentialsWithoutChangingProfile(t *testing.T) {
	f := newAPIFixture(t)
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	const secret = "synthetic-catalog-secret"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer "+secret || r.ContentLength != 0 {
			t.Error("unexpected provider catalog request")
		}
		fmt.Fprint(w, `{"data":[{"id":"available-model","name":"Available Model"}]}`)
	}))
	defer provider.Close()
	ctx := context.Background()
	if err := f.server.storeAIProvider(ctx, aiProviderConfig{ID: "catalog-provider", Provider: "openai", BaseURL: provider.URL + "/v1", Model: "selected-model", APIKey: secret, Enabled: true}, "admin"); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodPost, "/api/v1/admin/ai-provider/models", map[string]string{"id": "catalog-provider"})
	if response.StatusCode != http.StatusOK || bytes.Contains(body, []byte(secret)) {
		t.Fatalf("unexpected model catalog response: %d %s", response.StatusCode, body)
	}
	payload := decodeMap(t, body)
	models := payload["models"].([]any)
	if len(models) != 1 || models[0].(map[string]any)["id"] != "available-model" || payload["capabilities_verified"] != false {
		t.Fatalf("unexpected catalog: %s", body)
	}
	cfg, sealed, err := f.server.loadAIProviderByID(ctx, "catalog-provider")
	if err != nil || cfg.Model != "selected-model" || sealed.LastTestAt != nil {
		t.Fatalf("model catalog modified provider settings or test status")
	}
}
