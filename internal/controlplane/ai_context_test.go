package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAIReviewReusesNativePoolAndManualEvidenceWithoutMutatingIt(t *testing.T) {
	f := newAPIFixture(t)
	desired := map[string]any{"event": "ReviewedEvent", "password": "private-desired-value"}
	input := assuranceRequest{Raw: "event=Observed password=private-raw-value", Service: "custom", Desired: desired, FieldMapping: map[string]string{"event": "operator_chosen_label"}}
	result := map[string]any{
		"runtime_pool":           map[string]any{"builtin_profiles": []string{"baseline-mail"}, "managed_profiles": []string{"approved-addon"}, "pool_hash": "synthetic-pool"},
		"parser":                 map[string]any{"name": "baseline-mail"},
		"trace":                  []any{map[string]any{"parser": "approved-addon", "success": false}},
		"actual":                 map[string]any{"event": "Observed", "password": "private-actual-value"},
		"parser_creation_helper": map[string]any{"fields": []any{map[string]any{"name": "event", "source_backed": true}}},
	}
	before, _ := json.Marshal(result)
	prompt := f.server.aiReviewPrompt(input, result)
	for _, required := range []string{"approved-addon", "baseline-mail", "operator_chosen_label", "native_field_suggestions", "regression fixtures", "exact-client-bundle acknowledgement", "separate additive draft"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("missing native context %q", required)
		}
	}
	for _, secret := range []string{"private-desired-value", "private-raw-value", "private-actual-value"} {
		if strings.Contains(prompt, secret) {
			t.Fatal("prompt retained a private value")
		}
	}
	after, _ := json.Marshal(result)
	if string(before) != string(after) || desired["password"] != "private-desired-value" {
		t.Fatal("building a review prompt changed native evidence")
	}
	input.Raw = strings.Repeat("界", 40000)
	prompt = f.server.aiReviewPrompt(input, result)
	if !utf8.ValidString(prompt) || len(prompt) > 50<<10 || !strings.Contains(prompt, "approved-addon") || !strings.Contains(prompt, "truncated") {
		t.Fatal("large sample displaced native evidence or exceeded prompt bounds")
	}
}

func TestAIDiagnosticInvalidNativeSyntaxIsNotSaved(t *testing.T) {
	t.Setenv("NLI_AI_ALLOW_LOOPBACK", "true")
	f := newAPIFixture(t)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"name":"unsupported AI language","parser":{"pattern_type":"python","code":"return event"}}`}}}})
	}))
	defer provider.Close()
	if err := f.server.storeAIProvider(context.Background(), aiProviderConfig{Enabled: true, Provider: "openai", BaseURL: provider.URL, Model: "test", APIKey: "synthetic-provider-key"}, "admin"); err != nil {
		t.Fatal(err)
	}
	response, body := f.request(http.MethodPost, "/api/v1/assurance/ai-proposal", map[string]any{"raw": "status=failed", "service": "custom", "desired": map[string]any{"status": "expected-other-value"}})
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid language accepted: %d %s", response.StatusCode, body)
	}
	_, body = f.request(http.MethodGet, "/api/v1/assurance/ai-proposals", nil)
	if strings.Contains(string(body), "unsupported AI language") {
		t.Fatal("invalid AI parser persisted for review")
	}
	if err := validateAIParserSyntax(map[string]any{"pattern_type": "regex", "pattern": "(?<=x)y", "fields": []any{map[string]any{"name": "value", "source": "group:1", "type": "string"}}}); err == nil {
		t.Fatal("non-RE2 syntax accepted")
	}
	if err := validateAIParserSyntax(map[string]any{"pattern_type": "regex", "pattern": "(?P<value>.*)", "fields": []any{map[string]any{"name": "value", "source": "group:value", "type": "string"}}, "unknown_language": "custom"}); err == nil {
		t.Fatal("unknown parser property accepted")
	}
}

func TestAIReviewUsesFieldSenseForFailedNewLogAndHonorsContextBounds(t *testing.T) {
	f := newAPIFixture(t)
	result := map[string]any{"actual": map[string]any{}}
	input := assuranceRequest{Raw: "20250811 080053105+0100 aaa-gmailrgi009 smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed:reason=notneeded:server=cloud password=private-structural-preview-value", Service: "auto"}
	prompt := f.server.aiReviewPromptWithContext(context.Background(), input, result)
	for _, expected := range []string{"fieldsense_structural_analysis", "loglevel", "event", "reason", "review_required", "builtin_profiles", "smtp_native_v5"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("native FieldSense evidence missing %q", expected)
		}
	}
	if _, exists := result["fieldsense_analysis"]; exists {
		t.Fatal("AI context modified the original failed analysis")
	}
	if strings.Contains(prompt, "private-structural-preview-value") || strings.Contains(prompt, `"preview":"private-`) {
		t.Fatal("structural context retained a secret preview outside its keyed raw context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if prompt = f.server.aiReviewPromptWithContext(ctx, input, result); strings.Contains(prompt, "review_required") || !strings.Contains(prompt, "cancelled") {
		t.Fatal("cancelled context continued native discovery")
	}
	input.Raw = strings.Repeat("x", 65537)
	if prompt = f.server.aiReviewPromptWithContext(context.Background(), input, result); !strings.Contains(prompt, "larger supported samples") {
		t.Fatal("optional native AI context bound was not explicit")
	}
}
