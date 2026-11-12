package aiprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDiscoverGeminiModelsPagesTextGenerationAndExcludesCredential(t *testing.T) {
	const key = "synthetic-private-key"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/models" || r.Header.Get("x-goog-api-key") != key || r.ContentLength != 0 {
			t.Error("unexpected Gemini catalog request")
		}
		switch r.URL.Query().Get("pageToken") {
		case "":
			fmt.Fprint(w, `{"models":[{"name":"models/gemini-first","displayName":"Gemini First","supportedGenerationMethods":["generateContent"]},{"name":"models/embed","supportedGenerationMethods":["embedContent"]}],"nextPageToken":"next-token"}`)
		case "next-token":
			fmt.Fprintf(w, `{"models":[{"name":"models/%s","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-first","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-second","supportedGenerationMethods":["generateContent"]}]}`, key)
		default:
			t.Error("unexpected pagination cursor")
		}
	}))
	defer provider.Close()
	profile := Profile{ID: "gemini-discovery", Name: "Gemini", Kind: KindGemini, BaseURL: provider.URL + "/v1beta", Model: "gemini-first", Credential: Credential{Source: CredentialEnvironment, Reference: "NLI_DISCOVERY_TEST_KEY"}}
	got, err := DiscoverModels(context.Background(), profile, SecretResolver{Getenv: func(string) string { return key }}, ValidationPolicy{AllowLoopback: true})
	want := ModelCatalog{Models: []DiscoveredModel{{ID: "gemini-first", Label: "Gemini First"}, {ID: "gemini-second", Label: "gemini-second"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("catalog = %#v, error = %v, want %#v", got, err, want)
	}
}

func TestDiscoverModelsResponseAndEndpointLimits(t *testing.T) {
	const key = "synthetic-private-key"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("unexpected model catalog request")
		}
		fmt.Fprint(w, strings.Repeat("x", 1025))
	}))
	defer provider.Close()
	profile := Profile{ID: "test", Name: "Test", Kind: KindOpenAI, BaseURL: provider.URL + "/v1", Model: "some-model", MaxResponseBytes: 1024, Credential: Credential{Source: CredentialEnvironment, Reference: "NLI_DISCOVERY_TEST_KEY"}}
	resolver := SecretResolver{Getenv: func(string) string { return key }}
	if _, err := DiscoverModels(context.Background(), profile, resolver, ValidationPolicy{AllowLoopback: true}); err == nil || strings.Contains(err.Error(), key) {
		t.Fatalf("expected redacted response-limit error, got %v", err)
	}
	profile.Kind = KindOpenAICompatible
	if _, err := DiscoverModels(context.Background(), profile, resolver, ValidationPolicy{AllowLoopback: true}); !errors.Is(err, ErrModelDiscoveryUnsupported) {
		t.Fatalf("unsupported compatible provider returned %v", err)
	}
}
