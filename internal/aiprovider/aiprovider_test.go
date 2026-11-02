package aiprovider

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testAEAD(t *testing.T) cipher.AEAD {
	t.Helper()
	block, err := aes.NewCipher([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return aead
}

func validProfile(t *testing.T, id string, kind Kind, baseURL string) Profile {
	t.Helper()
	credential, err := SealInline(testAEAD(t), id, "test-secret-123456")
	if err != nil {
		t.Fatal(err)
	}
	return Profile{
		ID: id, Name: id, Kind: kind, BaseURL: baseURL, Model: "test-model",
		Enabled: true, Priority: 10, Credential: credential,
	}
}

func TestValidateKindsSecurityAndHeaders(t *testing.T) {
	for _, kind := range []Kind{KindOpenAI, KindNVIDIANIM, KindAnthropic, KindMiniMax, KindQwenDashScope, KindGemini, KindOpenAICompatible} {
		profile := validProfile(t, strings.ReplaceAll(string(kind), "_", "-"), kind, "https://provider.example/v1")
		if err := Validate(profile, ValidationPolicy{}); err != nil {
			t.Fatalf("kind %s should validate: %v", kind, err)
		}
	}
	ollama := Profile{ID: "ollama", Name: "Ollama", Kind: KindOllama, BaseURL: "http://127.0.0.1:11434", Model: "local", Enabled: true, Credential: Credential{Source: CredentialNone}}
	if err := Validate(ollama, ValidationPolicy{}); err != nil {
		t.Fatalf("controller-local Ollama should validate: %v", err)
	}
	bad := validProfile(t, "blocked", KindOpenAI, "http://169.254.169.254/latest")
	if err := Validate(bad, ValidationPolicy{}); err == nil {
		t.Fatal("metadata/link-local endpoint must be rejected")
	}
	bad = validProfile(t, "headers", KindOpenAI, "https://provider.example/v1")
	bad.Headers = map[string]string{"Authorization": "Bearer stolen"}
	if err := Validate(bad, ValidationPolicy{}); err == nil {
		t.Fatal("credential-bearing custom header must be rejected")
	}
	if err := ValidateResolvedIP(net.ParseIP("10.0.0.1"), ValidationPolicy{}); err == nil {
		t.Fatal("private IP must be rejected by default")
	}
}

func TestInlineEnvironmentAndFileCredentialResolution(t *testing.T) {
	aead := testAEAD(t)
	profile := validProfile(t, "inline", KindOpenAI, "https://provider.example/v1")
	secret, err := (SecretResolver{AEAD: aead}).Resolve(profile)
	if err != nil || secret != "test-secret-123456" {
		t.Fatalf("inline resolution failed: %q %v", secret, err)
	}

	profile.ID = "environment"
	profile.Credential = Credential{Source: CredentialEnvironment, Reference: "NLI_AI_TEST_KEY"}
	secret, err = (SecretResolver{Getenv: func(name string) string {
		if name == "NLI_AI_TEST_KEY" {
			return "environment-secret"
		}
		return ""
	}}).Resolve(profile)
	if err != nil || secret != "environment-secret" {
		t.Fatalf("environment resolution failed: %q %v", secret, err)
	}

	root := t.TempDir()
	path := filepath.Join(root, "provider.key")
	if err := os.WriteFile(path, []byte("file-secret-1234\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	profile.ID = "file"
	profile.Credential = Credential{Source: CredentialFile, Reference: "provider.key"}
	secret, err = (SecretResolver{SecretRoot: root}).Resolve(profile)
	if err != nil || secret != "file-secret-1234" {
		t.Fatalf("file resolution failed: %q %v", secret, err)
	}
	profile.Credential.Reference = filepath.Join("..", "outside.key")
	if _, err := (SecretResolver{SecretRoot: root}).Resolve(profile); err == nil {
		t.Fatal("path traversal must be rejected")
	}
}

func TestRedactedViewNeverContainsStoredSecretMaterial(t *testing.T) {
	profile := validProfile(t, "redacted", KindOpenAI, "https://provider.example/v1")
	view := Redacted(profile)
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	for _, forbidden := range []string{"test-secret", profile.Credential.Ciphertext, profile.Credential.Nonce} {
		if forbidden != "" && strings.Contains(text, forbidden) {
			t.Fatalf("redacted view leaked %q", forbidden)
		}
	}
	if view.CredentialLast4 != "3456" || !view.Configured {
		t.Fatalf("expected safe credential metadata, got %+v", view)
	}
}

func TestRegistryRevisionAndDeterministicRouting(t *testing.T) {
	registry := NewRegistry(ValidationPolicy{})
	a := validProfile(t, "alpha", KindOpenAI, "https://provider.example/v1")
	b := validProfile(t, "beta", KindAnthropic, "https://provider.example/v1")
	a.Priority, b.Priority = 20, 10
	storedA, err := registry.Put(a, 0, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Put(b, 0, "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Put(a, 0, "admin"); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	storedA.Enabled = false
	if _, err := registry.Put(storedA, storedA.Revision, "admin"); err != nil {
		t.Fatal(err)
	}
	route := registry.Route("")
	if len(route) != 1 || route[0].ID != "beta" {
		t.Fatalf("unexpected route: %+v", route)
	}
}

func TestStrictImportAndMaterialization(t *testing.T) {
	document := `{
		"id":"imported","name":"Imported","kind":"openai","base_url":"https://provider.example/v1",
		"model":"model","enabled":true,"priority":2,
		"credential":{"source":"encrypted_inline","secret":"import-secret-1234"}
	}`
	pending, err := ParseImport(strings.NewReader(document), DefaultMaxImportBytes)
	if err != nil {
		t.Fatal(err)
	}
	preview, _ := json.Marshal(pending.Preview())
	if strings.Contains(string(preview), "import-secret") {
		t.Fatal("import preview leaked plaintext secret")
	}
	profile, err := pending.Materialize(testAEAD(t), ValidationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Credential.Ciphertext == "" || strings.Contains(profile.Credential.Ciphertext, "import-secret") {
		t.Fatal("import was not sealed")
	}
	unknown := strings.Replace(document, `"priority":2,`, `"priority":2,"unexpected":true,`, 1)
	if _, err := ParseImport(strings.NewReader(unknown), DefaultMaxImportBytes); err == nil {
		t.Fatal("unknown import field must be rejected")
	}
	if _, err := ParseImport(strings.NewReader(document+` {}`), DefaultMaxImportBytes); err == nil {
		t.Fatal("trailing JSON must be rejected")
	}
}

func TestProviderSpecificConnectionRequests(t *testing.T) {
	tests := []struct {
		kind       Kind
		pathSuffix string
		header     string
	}{
		{KindOpenAI, "/chat/completions", "Authorization"},
		{KindNVIDIANIM, "/chat/completions", "Authorization"},
		{KindAnthropic, "/messages", "x-api-key"},
		{KindMiniMax, "/chat/completions", "Authorization"},
		{KindQwenDashScope, "/chat/completions", "Authorization"},
		{KindGemini, "/models/test-model:generateContent", "x-goog-api-key"},
		{KindOpenAICompatible, "/chat/completions", "Authorization"},
		{KindOllama, "/api/chat", ""},
	}
	for _, test := range tests {
		t.Run(string(test.kind), func(t *testing.T) {
			var requestError string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, test.pathSuffix) {
					requestError = "unexpected path: " + r.URL.Path
				}
				if test.header != "" && r.Header.Get(test.header) == "" {
					requestError = "missing authentication header " + test.header
				}
				body, _ := io.ReadAll(r.Body)
				if !json.Valid(body) {
					requestError = "request body is not JSON"
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer server.Close()
			id := strings.ReplaceAll(string(test.kind), "_", "-")
			profile := validProfile(t, id, test.kind, server.URL+"/v1")
			if test.kind == KindOllama {
				profile.BaseURL = server.URL
				profile.Credential = Credential{Source: CredentialNone}
			}
			result := (Tester{Policy: ValidationPolicy{AllowLoopback: true}, Resolver: SecretResolver{AEAD: testAEAD(t)}}).Test(context.Background(), profile)
			if !result.OK || result.ErrorCode != "" {
				t.Fatalf("connection test failed: %+v", result)
			}
			if requestError != "" {
				t.Fatal(requestError)
			}
		})
	}
}

func TestConnectionResultDoesNotExposeResponseOrCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"secret response detail"}`))
	}))
	defer server.Close()
	profile := validProfile(t, "failure", KindOpenAI, server.URL)
	result := (Tester{Policy: ValidationPolicy{AllowLoopback: true}, Resolver: SecretResolver{AEAD: testAEAD(t)}}).Test(context.Background(), profile)
	payload, _ := json.Marshal(result)
	if result.ErrorCode != "http_401" || strings.Contains(string(payload), "secret") {
		t.Fatalf("connection result was not sanitized: %s", payload)
	}
}

func TestTemplatesCoverAllKindsWithoutSecrets(t *testing.T) {
	wanted := map[Kind]bool{
		KindOpenAI: false, KindNVIDIANIM: false, KindAnthropic: false, KindMiniMax: false,
		KindQwenDashScope: false, KindGemini: false, KindOpenAICompatible: false, KindOllama: false,
	}
	for _, template := range Templates() {
		wanted[template.Kind] = true
		payload, _ := json.Marshal(template)
		if strings.Contains(strings.ToLower(string(payload)), "api_key") || strings.Contains(strings.ToLower(string(payload)), "secret") {
			t.Fatalf("template contains secret-shaped data: %s", payload)
		}
	}
	for kind, found := range wanted {
		if !found {
			t.Fatalf("missing template for %s", kind)
		}
	}
}
