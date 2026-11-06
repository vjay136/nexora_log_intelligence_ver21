package aiprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeRequiresUsableReplyAndDistinguishesReasoningAcknowledgement(t *testing.T) {
	for _, tc := range []struct {
		body    string
		ok, ack bool
		code    string
	}{{`{"ok":true}`, false, false, "invalid_protocol_response"}, {`{"choices":[{"message":{"reasoning_content":"private reasoning"},"finish_reason":"length"}]}`, false, true, "incomplete_output"}, {`{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}]}`, true, true, ""}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tc.body))
		}))
		profile := validProfile(t, "probe", KindNVIDIANIM, server.URL)
		result := (Tester{Policy: ValidationPolicy{AllowLoopback: true}, Resolver: SecretResolver{AEAD: testAEAD(t)}}).Test(context.Background(), profile)
		server.Close()
		if result.OK != tc.ok || result.Acknowledged != tc.ack || result.ErrorCode != tc.code {
			t.Fatalf("incorrect protocol classification: %+v", result)
		}
	}
}
func TestVisionStreamingProbeRequestIsExplicitBoundedAndKeepsCustomModel(t *testing.T) {
	profile := validProfile(t, "vision", KindNVIDIANIM, "https://integrate.api.nvidia.com/v1/chat/completions")
	profile.Model = "minimaxai/minimax-m3.1-flash"
	request, err := BuildRequestWithOptions(context.Background(), profile, "fake-key-value", "Describe image", RequestOptions{Stream: true, ImageURL: "https://assets.ngc.nvidia.com/products/example.jpg", MaxTokens: 64})
	if err != nil {
		t.Fatal(err)
	}
	if request.URL.Path != "/v1/chat/completions" || request.Header.Get("Accept") != "text/event-stream" {
		t.Fatal("full endpoint duplicated or streaming not requested")
	}
	var body map[string]any
	_ = json.NewDecoder(request.Body).Decode(&body)
	if body["model"] != profile.Model || body["stream"] != true || body["max_tokens"] != float64(64) {
		t.Fatal("custom model/probe settings changed")
	}
	encoded, _ := json.Marshal(body["messages"])
	if !strings.Contains(string(encoded), "image_url") {
		t.Fatal("vision content missing")
	}
	for _, value := range []string{"http://example.test/a.png", "https://127.0.0.1/a.png", "https://localhost/a.png", "https://public.example/a?secret=x"} {
		if ValidateImageProbeURL(value) == nil {
			t.Fatal("unsafe/private image probe URL accepted")
		}
	}
	for _, secret := range []string{"$NVIDIA_API_KEY", "${NVIDIA_API_KEY}", "Bearer fake-key-value", "YOUR_API_KEY"} {
		if _, err := SealInline(testAEAD(t), "placeholder", secret); err == nil {
			t.Fatal("credential placeholder accepted")
		}
	}
}
func TestRedactedProfileDoesNotExposeCustomHeaderValues(t *testing.T) {
	profile := validProfile(t, "headers", KindOpenAI, "https://example.test/v1")
	profile.Headers = map[string]string{"X-Internal-Custom": "private-header-value"}
	view := Redacted(profile)
	encoded, _ := json.Marshal(view)
	if strings.Contains(string(encoded), "private-header-value") {
		t.Fatal("header value leaked through redacted catalog")
	}
}
