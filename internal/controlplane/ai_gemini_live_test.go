package controlplane

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestGeminiLiveWorkspace(t *testing.T) {
	key := os.Getenv("NLI_TEST_GEMINI_API_KEY")
	if key == "" {
		t.Skip("temporary Gemini credential not configured")
	}
	f := newAPIFixture(t)
	err := f.server.storeAIProvider(context.Background(), aiProviderConfig{
		ID: "gemini-smoke", Provider: "gemini", Model: "gemini-3.5-flash-lite",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta",
		Enabled: true, APIKey: key, TimeoutSeconds: 15, MaxResponseBytes: 1048576,
	}, "admin")
	if err != nil {
		t.Fatal("isolated provider configuration failed")
	}
	for _, tc := range []struct {
		name string
		path string
		body map[string]any
	}{
		{"mapping", "/api/v1/field-mapping/ai-suggest", map[string]any{"raw": "20250811 080053105+0100 mail.example.test smtp 3692 0 140026854221568 Note;SieveRuleMsg status=failed.reason;notneeded,server:cloud", "service": "smtp", "native_fallback": true}},
		{"parser draft", "/api/v1/ai/workspace", map[string]any{"workspace": "building-blocks", "mode": "fix", "artifact": "parser", "raw": "smtp status=failed reason=queued", "native_fallback": true}},
		{"KV template", "/api/v1/ai/workspace", map[string]any{"workspace": "kv-templates", "mode": "fix", "artifact": "kv_template", "raw": "status=failed|reason=queued", "question": "Use the literal pipe as the pair separator and equals as the assignment operator."}},
		{"help", "/api/v1/ai/workspace", map[string]any{"workspace": "quarantine-wizard", "tab": "Step 4", "mode": "ask", "raw": "smtp status=failed", "question": "Which NLI selector extracts status? Give the source key and one next step."}},
		{"regex draft", "/api/v1/ai/workspace", map[string]any{"workspace": "regex", "mode": "fix", "artifact": "regex", "raw": "status=failed", "question": "Create a Go RE2 regex covering the entire sample with named capture status."}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := f.request(http.MethodPost, tc.path, tc.body)
			result := decodeMap(t, body)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("controller HTTP %d; error code %v", resp.StatusCode, result["error_code"])
			}
			if result["activated"] != false || result["persisted"] != false || result["provider_id"] != "gemini-smoke" {
				t.Fatal("unexpected response metadata or provider")
			}
			if tc.name == "mapping" || tc.name == "parser draft" {
				validation := result["validation"].(map[string]any)
				if result["proposal_origin"] != "ai" || validation["extraction_valid"] != true || validation["actual"].(map[string]any)["status"] != "failed" {
					t.Fatalf("mapping not verified as Gemini origin: %v", result["proposal_origin"])
				}
				if tc.name == "mapping" {
					actual := validation["actual"].(map[string]any)
					if actual["reason"] != "notneeded" || actual["server"] != "cloud" {
						t.Fatal("mixed-delimiter fields were not verified")
					}
				}
			} else if tc.name == "regex draft" || tc.name == "KV template" {
				if result["proposal"] == nil || result["validation"] == nil {
					t.Fatal("missing verified regex proposal")
				}
			} else if answer, _ := result["answer"].(string); !strings.Contains(strings.ToLower(answer), "status") {
				t.Fatal("empty Gemini guidance")
			}
		})
	}
}
