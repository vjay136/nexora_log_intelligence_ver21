package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/aiprovider"
	"nexora.local/nli-v20/internal/familyprofile"
	"nexora.local/nli-v20/internal/middleware"
	"nexora.local/nli-v20/internal/registry"
	metastore "nexora.local/nli-v20/internal/store"
)

type aiProviderConfig struct {
	Provider       string                `json:"provider"`
	BaseURL        string                `json:"base_url"`
	Model          string                `json:"model"`
	APIKey         string                `json:"-"`
	LegacyAPIKey   string                `json:"api_key,omitempty"`
	ID             string                `json:"id,omitempty"`
	Name           string                `json:"name,omitempty"`
	Enabled        bool                  `json:"enabled"`
	Priority       int                   `json:"priority,omitempty"`
	FallbackGroup  string                `json:"fallback_group,omitempty"`
	Credential     aiprovider.Credential `json:"credential,omitempty"`
	Headers        map[string]string     `json:"headers,omitempty"`
	TimeoutSeconds int                   `json:"timeout_seconds,omitempty"`
}
type sealedAIProvider struct {
	Nonce        string     `json:"nonce"`
	Ciphertext   string     `json:"ciphertext"`
	ConfiguredAt time.Time  `json:"configured_at"`
	UpdatedBy    string     `json:"updated_by"`
	LastTestAt   *time.Time `json:"last_test_at,omitempty"`
}

func (s *Server) adminOnly(w http.ResponseWriter, r *http.Request) *middleware.SecurityContext {
	p := s.authorize(w, r, middleware.PermissionManageUsers)
	if p == nil {
		return nil
	}
	if p.Role != "admin" {
		jsonError(w, "administrator role required", http.StatusForbidden)
		return nil
	}
	return p
}
func (s *Server) storeAIProvider(ctx context.Context, cfg aiProviderConfig, actor string) error {
	if s.assurance.providerAEAD == nil {
		return errors.New("Assurance encryption key is not configured")
	}
	cfg = normalizeProvider(cfg)
	if cfg.APIKey == "" {
		cfg.APIKey = cfg.LegacyAPIKey
	}
	if cfg.APIKey != "" {
		credential, err := aiprovider.SealInline(s.assurance.providerAEAD, cfg.ID, cfg.APIKey)
		if err != nil {
			return err
		}
		cfg.Credential = credential
	}
	cfg.LegacyAPIKey = ""
	cfg.APIKey = ""
	if err := validateProviderWithPolicy(cfg, providerPolicy()); err != nil {
		return err
	}
	plain, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	nonce := make([]byte, s.assurance.providerAEAD.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	sealed := sealedAIProvider{Nonce: base64.StdEncoding.EncodeToString(nonce), Ciphertext: base64.StdEncoding.EncodeToString(s.assurance.providerAEAD.Seal(nil, nonce, plain, []byte("ai-provider"))), ConfiguredAt: time.Now().UTC(), UpdatedBy: actor}
	payload, _ := json.Marshal(sealed)
	_, err = s.metadata.Put(ctx, metastore.Document{Collection: "ai-provider", ID: "default", State: "configured", Payload: payload, CreatedBy: actor})
	return err
}
func (s *Server) loadAIProvider(ctx context.Context) (aiProviderConfig, sealedAIProvider, error) {
	if s.assurance.providerAEAD == nil {
		return aiProviderConfig{}, sealedAIProvider{}, errors.New("Assurance encryption key is not configured")
	}
	doc, err := s.metadata.Get(ctx, "ai-provider", "default", 0)
	if err != nil {
		return aiProviderConfig{}, sealedAIProvider{}, err
	}
	var sealed sealedAIProvider
	if err = json.Unmarshal(doc.Payload, &sealed); err != nil {
		return aiProviderConfig{}, sealed, err
	}
	nonce, err := base64.StdEncoding.DecodeString(sealed.Nonce)
	if err != nil || len(nonce) != s.assurance.providerAEAD.NonceSize() {
		return aiProviderConfig{}, sealed, errors.New("invalid provider nonce")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(sealed.Ciphertext)
	if err != nil {
		return aiProviderConfig{}, sealed, err
	}
	plain, err := s.assurance.providerAEAD.Open(nil, nonce, ciphertext, []byte("ai-provider"))
	if err != nil {
		return aiProviderConfig{}, sealed, err
	}
	var cfg aiProviderConfig
	if err = json.Unmarshal(plain, &cfg); err != nil {
		return cfg, sealed, err
	}
	cfg = normalizeProvider(cfg)
	if cfg.LegacyAPIKey != "" {
		cfg.APIKey = cfg.LegacyAPIKey
		return cfg, sealed, nil
	}
	// Secret references are resolved only when the provider is invoked. This
	// keeps the redacted status/catalog endpoint available when a referenced
	// environment variable or mounted secret is temporarily unavailable.
	return cfg, sealed, nil
}
func providerPolicy() aiprovider.ValidationPolicy {
	root := strings.TrimSpace(os.Getenv("NLI_AI_SECRET_ROOT"))
	if root == "" {
		root = "/etc/nli/ai-secrets"
	}
	return aiprovider.ValidationPolicy{SecretRoot: root, AllowLoopback: strings.EqualFold(strings.TrimSpace(os.Getenv("NLI_AI_ALLOW_LOOPBACK")), "true")}
}

func providerKind(value string) aiprovider.Kind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "custom", "openai-compatible", "openai_compatible":
		return aiprovider.KindOpenAICompatible
	case "nvidia", "nvidia-nim", "nvidia_nim":
		return aiprovider.KindNVIDIANIM
	case "qwen", "qwen-dashscope", "qwen_dashscope":
		return aiprovider.KindQwenDashScope
	default:
		return aiprovider.Kind(strings.ToLower(strings.TrimSpace(value)))
	}
}

func profileFromConfig(cfg aiProviderConfig) aiprovider.Profile {
	return aiprovider.Normalize(aiprovider.Profile{ID: cfg.ID, Name: cfg.Name, Kind: providerKind(cfg.Provider), BaseURL: cfg.BaseURL, Model: cfg.Model, Enabled: cfg.Enabled, Priority: cfg.Priority, FallbackGroup: cfg.FallbackGroup, TimeoutSeconds: cfg.TimeoutSeconds, Headers: cfg.Headers, Credential: cfg.Credential})
}

func validateProvider(cfg aiProviderConfig) error {
	policy := providerPolicy()
	policy.AllowLoopback = true // retained for isolated protocol-adapter tests
	return validateProviderWithPolicy(cfg, policy)
}

func validateProviderWithPolicy(cfg aiProviderConfig, policy aiprovider.ValidationPolicy) error {
	cfg = normalizeProvider(cfg)
	profile := profileFromConfig(cfg)
	if profile.Credential.Source == "" {
		if profile.Kind == aiprovider.KindOllama && cfg.APIKey == "" && cfg.LegacyAPIKey == "" {
			profile.Credential = aiprovider.Credential{Source: aiprovider.CredentialNone}
		} else {
			secret := cfg.APIKey
			if secret == "" {
				secret = cfg.LegacyAPIKey
			}
			if len(secret) < 8 || len(secret) > 16*1024 || strings.ContainsAny(secret, "\r\n\x00") {
				return errors.New("provider credential is invalid")
			}
			profile.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: "NLI_AI_VALIDATION_PLACEHOLDER"}
		}
	}
	return aiprovider.Validate(profile, policy)
}
func normalizeProvider(cfg aiProviderConfig) aiProviderConfig {
	cfg.Provider = string(providerKind(cfg.Provider))
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	if cfg.ID == "" {
		cfg.ID = "default"
	}
	if cfg.Name == "" {
		cfg.Name = "Default AI provider"
	}
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		for _, template := range aiprovider.Templates() {
			if template.Kind == providerKind(cfg.Provider) && template.Kind != aiprovider.KindOpenAICompatible {
				cfg.BaseURL = template.BaseURL
				break
			}
		}
	}
	return cfg
}
func (s *Server) publishAIProviderStatus(status, provider, model string) {
	payload, _ := json.Marshal(map[string]any{"type": "ai_provider_changed", "status": status, "provider": provider, "model": model, "updated_at": time.Now().UTC()})
	s.hub.Publish(payload)
}
func providerResponse(cfg aiProviderConfig, sealed sealedAIProvider) map[string]any {
	view := aiprovider.Redacted(profileFromConfig(cfg))
	if cfg.Credential.Source == "" && cfg.APIKey != "" {
		view.Configured = true
		view.CredentialSource = aiprovider.CredentialEncryptedInline
		if len(cfg.APIKey) >= 4 {
			view.CredentialLast4 = cfg.APIKey[len(cfg.APIKey)-4:]
		}
	}
	return map[string]any{
		"id": view.ID, "name": view.Name, "provider": string(view.Kind), "kind": string(view.Kind),
		"model": view.Model, "endpoint": view.BaseURL, "configured": view.Configured,
		"enabled": view.Enabled, "status": "configured", "key_last4": view.CredentialLast4,
		"credential_source": view.CredentialSource, "credential_ref": view.CredentialRef,
		"priority": view.Priority, "fallback_group": view.FallbackGroup, "last_test_at": sealed.LastTestAt,
	}
}
func providerCatalogResponse(provider map[string]any) map[string]any {
	return map[string]any{"provider": provider, "templates": aiprovider.Templates(), "credential_modes": []string{"encrypted_inline", "env_ref", "file_ref", "none"}}
}
func (s *Server) handleAIProvider(w http.ResponseWriter, r *http.Request) {
	p := s.adminOnly(w, r)
	if p == nil {
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, sealed, err := s.loadAIProvider(r.Context())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || s.assurance.providerAEAD == nil {
				jsonOK(w, providerCatalogResponse(map[string]any{"provider": "", "model": "", "endpoint": "", "configured": false, "key_last4": "", "enabled": false, "status": "not_configured", "last_test_at": nil, "credential_source": "encrypted_inline"}))
				return
			}
			jsonError(w, "provider store unavailable", 503)
			return
		}
		jsonOK(w, providerCatalogResponse(providerResponse(cfg, sealed)))
	case http.MethodPut:
		var request struct {
			Provider         string            `json:"provider"`
			Kind             string            `json:"kind"`
			ID               string            `json:"id"`
			Name             string            `json:"name"`
			BaseURL          string            `json:"base_url"`
			Endpoint         string            `json:"endpoint"`
			Model            string            `json:"model"`
			APIKey           string            `json:"api_key"`
			CredentialSource string            `json:"credential_source"`
			CredentialRef    string            `json:"credential_ref"`
			Priority         int               `json:"priority"`
			FallbackGroup    string            `json:"fallback_group"`
			Headers          map[string]string `json:"headers"`
			Enabled          *bool             `json:"enabled"`
		}
		if decodeJSON(r, &request, 16<<10) != nil {
			jsonError(w, "invalid provider configuration", 400)
			return
		}
		if request.Enabled != nil && !*request.Enabled {
			_ = s.metadata.Delete(r.Context(), "ai-provider", "default", false, p.Username)
			s.auditAction(p, "ai_provider.disable", "ai-provider/default", "success", nil)
			s.publishAIProviderStatus("disabled", "", "")
			jsonOK(w, map[string]any{"configured": false, "enabled": false, "encrypted": false})
			return
		}
		kind := request.Kind
		if kind == "" {
			kind = request.Provider
		}
		cfg := aiProviderConfig{Provider: kind, ID: request.ID, Name: request.Name, BaseURL: request.BaseURL, Model: request.Model, APIKey: request.APIKey, Enabled: true, Priority: request.Priority, FallbackGroup: request.FallbackGroup, Headers: request.Headers}
		if cfg.BaseURL == "" {
			cfg.BaseURL = request.Endpoint
		}
		cfg = normalizeProvider(cfg)
		if request.CredentialSource == "" {
			request.CredentialSource = string(aiprovider.CredentialEncryptedInline)
		}
		if cfg.APIKey == "" && request.CredentialSource == string(aiprovider.CredentialEncryptedInline) {
			if previous, _, err := s.loadAIProvider(r.Context()); err == nil {
				if previous.Credential.Source == aiprovider.CredentialEncryptedInline {
					cfg.Credential = previous.Credential
				} else if previous.APIKey != "" {
					// Transparently migrate the legacy encrypted record to the
					// provider-neutral inline credential envelope on the next save.
					cfg.APIKey = previous.APIKey
				}
			}
		} else if request.CredentialSource == string(aiprovider.CredentialEnvironment) {
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: request.CredentialRef}
		} else if request.CredentialSource == string(aiprovider.CredentialFile) {
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialFile, Reference: request.CredentialRef}
		} else if request.CredentialSource == string(aiprovider.CredentialNone) {
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialNone}
		}
		if err := validateProviderWithPolicy(cfg, providerPolicy()); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if err := s.storeAIProvider(r.Context(), cfg, p.Username); err != nil {
			jsonError(w, "provider configuration could not be encrypted or stored", 503)
			return
		}
		s.auditAction(p, "ai_provider.configure", "ai-provider/default", "success", map[string]string{"provider": cfg.Provider, "model": cfg.Model})
		s.publishAIProviderStatus("configured", cfg.Provider, cfg.Model)
		stored, sealed, err := s.loadAIProvider(r.Context())
		if err != nil {
			jsonError(w, "provider configuration could not be read after storage", 503)
			return
		}
		jsonOK(w, providerCatalogResponse(providerResponse(stored, sealed)))
	default:
		jsonError(w, "GET or PUT required", 405)
	}
}
func (s *Server) callAI(ctx context.Context, cfg aiProviderConfig, prompt string) (string, error) {
	policy := providerPolicy()
	policy.AllowLoopback = true // compatibility wrapper for isolated adapter tests
	return s.callAIWithPolicy(ctx, cfg, prompt, policy)
}
func (s *Server) callAIWithPolicy(ctx context.Context, cfg aiProviderConfig, prompt string, policy aiprovider.ValidationPolicy) (string, error) {
	cfg = normalizeProvider(cfg)
	if err := validateProviderWithPolicy(cfg, policy); err != nil {
		return "", err
	}
	if cfg.APIKey == "" && cfg.Credential.Source != "" {
		secret, err := (aiprovider.SecretResolver{AEAD: s.assurance.providerAEAD, SecretRoot: policy.SecretRoot}).Resolve(profileFromConfig(cfg))
		if err != nil {
			return "", err
		}
		cfg.APIKey = secret
	}
	systemPrompt := "Return only a JSON object with keys name, description, parser, regex, template, kv, field_mapping, and projection. parser must be declarative extraction metadata only. Suggestions must be JSON data for their named category. Never include executable code, shell commands, secrets, or extra sample logs."
	req, err := aiprovider.BuildRequestWithOptions(ctx, profileFromConfig(cfg), cfg.APIKey, prompt, aiprovider.RequestOptions{SystemPrompt: systemPrompt, MaxTokens: 2048})
	if err != nil {
		return "", err
	}
	if providerKind(cfg.Provider) == aiprovider.KindOllama {
		policy.AllowLoopback = true
	}
	client := aiprovider.NewSafeHTTPClient(policy, 15*time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if len(data) >= 1<<20 {
		return "", errors.New("provider response exceeded 1 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	if providerKind(cfg.Provider) == aiprovider.KindAnthropic {
		var envelope struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}
		if json.Unmarshal(data, &envelope) != nil {
			return "", errors.New("Anthropic provider returned an invalid response")
		}
		for _, block := range envelope.Content {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				return block.Text, nil
			}
		}
		return "", errors.New("Anthropic provider returned no text content")
	}
	if providerKind(cfg.Provider) == aiprovider.KindGemini {
		var envelope struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if json.Unmarshal(data, &envelope) != nil || len(envelope.Candidates) == 0 || len(envelope.Candidates[0].Content.Parts) == 0 || strings.TrimSpace(envelope.Candidates[0].Content.Parts[0].Text) == "" {
			return "", errors.New("Gemini provider returned an invalid response")
		}
		return envelope.Candidates[0].Content.Parts[0].Text, nil
	}
	if providerKind(cfg.Provider) == aiprovider.KindOllama {
		var envelope struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(data, &envelope) != nil || strings.TrimSpace(envelope.Message.Content) == "" {
			return "", errors.New("Ollama provider returned an invalid response")
		}
		return envelope.Message.Content, nil
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return "", errors.New("OpenAI-compatible provider returned an invalid response")
	}
	return envelope.Choices[0].Message.Content, nil
}
func (s *Server) handleAIProviderTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", 405)
		return
	}
	p := s.adminOnly(w, r)
	if p == nil {
		return
	}
	var input struct {
		Provider         string `json:"provider"`
		Kind             string `json:"kind"`
		ID               string `json:"id"`
		Name             string `json:"name"`
		Model            string `json:"model"`
		Endpoint         string `json:"endpoint"`
		BaseURL          string `json:"base_url"`
		APIKey           string `json:"api_key"`
		CredentialSource string `json:"credential_source"`
		CredentialRef    string `json:"credential_ref"`
	}
	if r.ContentLength > 0 && decodeJSON(r, &input, 16<<10) != nil {
		jsonError(w, "invalid provider test request", 400)
		return
	}
	cfg, _, loadErr := s.loadAIProvider(r.Context())
	if loadErr != nil {
		cfg = aiProviderConfig{Enabled: true}
	}
	if input.Kind != "" {
		cfg.Provider = input.Kind
	} else if input.Provider != "" {
		cfg.Provider = input.Provider
	}
	if input.ID != "" {
		cfg.ID = input.ID
	}
	if input.Name != "" {
		cfg.Name = input.Name
	}
	if input.Model != "" {
		cfg.Model = input.Model
	}
	if input.BaseURL != "" {
		cfg.BaseURL = input.BaseURL
	} else if input.Endpoint != "" {
		cfg.BaseURL = input.Endpoint
	}
	cfg = normalizeProvider(cfg)
	if input.APIKey != "" {
		credential, sealErr := aiprovider.SealInline(s.assurance.providerAEAD, cfg.ID, input.APIKey)
		if sealErr != nil {
			jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "Provider credential is invalid"})
			return
		}
		cfg.Credential = credential
	} else if input.CredentialSource == string(aiprovider.CredentialEnvironment) {
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: input.CredentialRef}
	} else if input.CredentialSource == string(aiprovider.CredentialFile) {
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialFile, Reference: input.CredentialRef}
	} else if input.CredentialSource == string(aiprovider.CredentialNone) {
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialNone}
	}
	if loadErr != nil && cfg.Credential.Source == "" {
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "AI provider is not configured"})
		return
	}
	policy := providerPolicy()
	if err := aiprovider.Validate(profileFromConfig(cfg), policy); err != nil {
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "Provider URL, model or key is invalid"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 17*time.Second)
	defer cancel()
	result := (aiprovider.Tester{Policy: policy, Resolver: aiprovider.SecretResolver{AEAD: s.assurance.providerAEAD, SecretRoot: policy.SecretRoot}}).Test(ctx, profileFromConfig(cfg))
	if !result.OK {
		s.auditAction(p, "ai_provider.test", "ai-provider/default", "failure", nil)
		s.publishAIProviderStatus("connection_failed", cfg.Provider, cfg.Model)
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "Provider connection failed", "error_code": result.ErrorCode, "status_code": result.StatusCode})
		return
	}
	if stored, _, storedErr := s.loadAIProvider(r.Context()); storedErr == nil && stored.ID == cfg.ID && stored.Credential == cfg.Credential {
		_ = s.markProviderTested(r.Context(), p.Username)
	}
	s.auditAction(p, "ai_provider.test", "ai-provider/default", "success", nil)
	s.publishAIProviderStatus("connected", cfg.Provider, cfg.Model)
	jsonOK(w, map[string]any{"connected": true, "status": "success", "message": "Provider connection succeeded", "model": cfg.Model, "status_code": result.StatusCode, "latency_ms": result.Latency.Milliseconds()})
}

func (s *Server) markProviderTested(ctx context.Context, actor string) error {
	doc, err := s.metadata.Get(ctx, "ai-provider", "default", 0)
	if err != nil {
		return err
	}
	var sealed sealedAIProvider
	if json.Unmarshal(doc.Payload, &sealed) != nil {
		return errors.New("invalid provider record")
	}
	now := time.Now().UTC()
	sealed.LastTestAt = &now
	payload, _ := json.Marshal(sealed)
	_, err = s.metadata.Put(ctx, metastore.Document{Collection: "ai-provider", ID: "default", State: "configured", Payload: payload, CreatedBy: actor})
	return err
}
func (s *Server) handleAssuranceAIProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", 405)
		return
	}
	var input assuranceRequest
	if decodeJSON(r, &input, 64<<10) != nil || strings.TrimSpace(input.Raw) == "" {
		jsonError(w, "raw input is required", 400)
		return
	}
	result := s.runAssurance(input)
	if result["passed"] == true {
		jsonError(w, "AI proposal is only available when deterministic assurance fails", 409)
		return
	}
	cfg, _, err := s.loadAIProvider(r.Context())
	if err != nil {
		jsonError(w, "AI provider is not configured or encryption is unavailable", 503)
		return
	}
	redacted := s.assuranceDeps.Masker.MaskForTemplate(input.Raw)
	prompt := fmt.Sprintf("Propose a declarative parser for this log. Return JSON only. Known parsed fields: %v. Expected fields: %v. Validation errors: %v. Log sample: %s", result["actual"], input.Desired, result["errors"], redacted)
	ctx, cancel := context.WithTimeout(r.Context(), 17*time.Second)
	defer cancel()
	aiStarted := time.Now()
	aiMetricsSuccess := false
	defer func() { s.aiMetrics.record(aiMetricsSuccess, time.Since(aiStarted)) }()
	raw, err := s.callAIWithPolicy(ctx, cfg, prompt, providerPolicy())
	if err != nil {
		jsonError(w, "AI parser proposal failed; no parser was changed", 502)
		return
	}
	var proposal map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(raw)), &proposal) != nil {
		jsonError(w, "AI returned an invalid parser proposal", 502)
		return
	}
	parser, _ := proposal["parser"].(map[string]any)
	if parser == nil {
		jsonError(w, "AI returned no declarative parser", 502)
		return
	}
	if err := unsafeDeclarative(parser); err != nil {
		jsonError(w, "AI proposal failed declarative safety validation", 422)
		return
	}
	categoryAliases := map[string][]string{
		"parser": {"parser"}, "regex": {"regex", "regex_pattern"}, "template": {"template"},
		"kv": {"kv", "key_value", "key-value"}, "field_mapping": {"field_mapping", "mapping"},
		"projection": {"projection", "projections"},
	}
	suggestions := map[string]any{}
	for category, aliases := range categoryAliases {
		for _, alias := range aliases {
			value, exists := proposal[alias]
			if !exists || value == nil {
				continue
			}
			encoded, marshalErr := json.Marshal(value)
			if marshalErr != nil || len(encoded) > 64<<10 {
				jsonError(w, "AI category suggestion is invalid or oversized", 422)
				return
			}
			var normalized any
			if json.Unmarshal(encoded, &normalized) != nil {
				continue
			}
			if object, ok := normalized.(map[string]any); ok {
				if err := unsafeDeclarative(object); err != nil {
					jsonError(w, "AI category suggestion failed declarative safety validation", 422)
					return
				}
				if category == "regex" {
					pattern, _ := object["pattern"].(string)
					if pattern != "" {
						if _, compileErr := regexp.Compile(pattern); compileErr != nil {
							jsonError(w, "AI regex suggestion is invalid", 422)
							return
						}
					}
				}
			}
			suggestions[category] = normalized
			break
		}
	}
	proposal["review_state"] = "draft"
	proposal["requires_human_approval"] = true
	proposal["source_hash"] = result["raw_sha256"]
	diagnosis := map[string]any{
		"summary":  "The deterministic parser did not satisfy the requested field and validation contract.",
		"score":    result["score"],
		"coverage": result["coverage"],
		"errors":   result["errors"],
		"parser":   result["parser"],
	}
	suggestedMethod := strings.TrimSpace(fmt.Sprint(proposal["description"]))
	if suggestedMethod == "" || suggestedMethod == "<nil>" {
		suggestedMethod = strings.TrimSpace(fmt.Sprint(proposal["name"]))
	}
	if suggestedMethod == "" || suggestedMethod == "<nil>" {
		suggestedMethod = "Create a bounded declarative parser, validate it against the failing case and regression corpus, then require independent approval before publication."
	}
	// Test inputs and examples often contain customer data. Persist only the
	// declarative parser body plus the input hash; never retain prompt samples.
	storedParser := make(map[string]any, len(parser))
	for key, value := range parser {
		if key == "regression_cases" || key == "tests" || key == "samples" || key == "examples" {
			continue
		}
		storedParser[key] = value
	}
	suggestions["parser"] = storedParser
	proposalID := safeID("proposal-" + fmt.Sprint(result["raw_sha256"]))
	if proposalID == "" {
		jsonError(w, "AI proposal identifier could not be created", 500)
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"name":                    proposal["name"],
		"proposal":                storedParser,
		"suggestions":             suggestions,
		"diagnosis":               diagnosis,
		"suggested_method":        suggestedMethod,
		"source_hash":             result["raw_sha256"],
		"review_state":            "draft",
		"requires_human_approval": true,
		"published":               false,
	})
	principal := middleware.GetSecurityContext(r)
	createdBy := "ai-proposal"
	if principal != nil {
		createdBy = principal.Username
	}
	saved, saveErr := s.metadata.Put(r.Context(), metastore.Document{Collection: "ai-proposals", ID: proposalID, State: "draft", Payload: payload, CreatedBy: createdBy})
	if saveErr != nil {
		jsonError(w, "AI proposal could not be saved for review", 503)
		return
	}
	s.auditAction(principal, "ai-proposal.create", "ai-proposals/"+saved.ID, "success", map[string]string{"state": "draft", "source_hash": fmt.Sprint(result["raw_sha256"])})
	event, _ := json.Marshal(map[string]any{"type": "ai_proposal_changed", "id": saved.ID, "status": "draft", "source_hash": result["raw_sha256"], "updated_at": time.Now().UTC()})
	s.hub.Publish(event)
	proposal["id"] = saved.ID
	proposal["diagnosis"] = diagnosis
	proposal["suggested_method"] = suggestedMethod
	aiMetricsSuccess = true
	jsonOK(w, map[string]any{"id": saved.ID, "proposal": proposal, "suggestions": suggestions, "categories": []string{"parser", "regex", "template", "kv", "field_mapping", "projection"}, "review_state": "draft", "requires_human_approval": true, "published": false, "stored": true})
}

type assuranceParserInput struct {
	ID                       string             `json:"id"`
	Name                     string             `json:"name"`
	Version                  string             `json:"version"`
	Parser                   map[string]any     `json:"parser"`
	Cases                    []assuranceRequest `json:"cases"`
	SourceQuarantineRecordID string             `json:"source_quarantine_record_id,omitempty"`
}

// handleAssuranceParserPreview validates a proposed declarative parser against
// the exact supplied evidence without persisting or activating it.  The QRW
// uses this endpoint before a draft exists so that its Test step exercises the
// candidate under review rather than whichever parser happens to be live.
func (s *Server) handleAssuranceParserPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var input assuranceParserInput
	if decodeJSON(r, &input, 2<<20) != nil || input.Parser == nil || len(input.Cases) == 0 || len(input.Cases) > 100 {
		jsonError(w, "declarative parser and 1..100 validation cases are required", http.StatusBadRequest)
		return
	}
	if err := unsafeDeclarative(input.Parser); err != nil {
		jsonError(w, "unsafe declarative parser", http.StatusBadRequest)
		return
	}
	validation := validateParserCandidate(input.Parser, input.Cases)
	jsonOK(w, map[string]any{
		"validation":        validation,
		"artifact_checksum": parserArtifactChecksum(input.Parser),
		"passed":            validation["failed"] == 0 && validation["total"] == len(input.Cases),
		"persisted":         false,
		"activated":         false,
	})
}

func validateParserCandidate(definition map[string]any, cases []assuranceRequest) map[string]any {
	encoded, _ := json.Marshal(definition)
	var parser registry.ParserDef
	if err := json.Unmarshal(encoded, &parser); err != nil {
		return map[string]any{"status": "invalid_definition", "reason": "parser definition is invalid", "total": len(cases), "passed": 0, "failed": len(cases), "score": 0, "errors": []string{"parser definition is invalid"}}
	}
	// Candidate editors intentionally do not assign the durable catalog identity
	// or revision. Supply bounded ephemeral values for the shared registry
	// validator; these values are not persisted and are excluded from the
	// artifact checksum that later binds review and publication.
	if strings.TrimSpace(parser.ID) == "" {
		parser.ID = "assurance-preview"
	}
	if parser.Version < 1 {
		parser.Version = 1
	}
	if err := registry.Validate(&parser); err != nil {
		return map[string]any{"status": "invalid_definition", "reason": err.Error(), "total": len(cases), "passed": 0, "failed": len(cases), "score": 0, "errors": []string{err.Error()}}
	}
	if len(cases) == 0 && len(parser.Tests) > 0 {
		for _, test := range parser.Tests {
			cases = append(cases, assuranceRequest{Raw: test.Input, Desired: test.Expected})
		}
	}
	if parser.PatternType == "mapping" {
		if _, _, _, err := nfiMappingScope(definition, *parser.MappingPlan, cases); err != nil {
			return map[string]any{"status": "invalid_definition", "reason": err.Error(), "total": len(cases), "passed": 0, "failed": len(cases), "score": 0, "errors": []string{err.Error()}}
		}
	}
	passed := 0
	outcomes := make([]map[string]any, 0, len(cases))
	for index, item := range cases {
		if strings.TrimSpace(item.Raw) == "" {
			outcomes = append(outcomes, map[string]any{"index": index, "passed": false, "actual": map[string]any{}, "diff": []map[string]any{}, "errors": []string{"validation input is empty"}})
			continue
		}
		actual, errors := registry.Apply(&parser, item.Raw)
		diff, _ := structuralDiff(actual, item.Desired)
		ok := len(errors) == 0 && len(diff) == 0 && len(item.Desired) > 0
		if ok {
			passed++
		}
		caseErrors := append([]string(nil), errors...)
		if len(item.Desired) == 0 {
			caseErrors = append(caseErrors, "expected output is empty")
		}
		outcomes = append(outcomes, map[string]any{"index": index, "passed": ok, "actual": actual, "diff": diff, "errors": caseErrors, "raw_sha256": fmt.Sprintf("%x", hashBytes([]byte(item.Raw)))})
	}
	total := len(cases)
	score := 0
	if total > 0 {
		score = passed * 100 / total
	}
	status, reason := "failed", "one or more regression cases did not match the expected output"
	if total == 0 {
		status, reason = "no_cases", "add at least one non-empty input and expected output"
	} else if passed == total {
		status, reason = "passed", "all regression cases matched"
	}
	return map[string]any{"status": status, "reason": reason, "total": total, "passed": passed, "failed": total - passed, "score": score, "results": outcomes}
}

// parserArtifactChecksum binds review and deployment to the exact declarative
// parser submitted by the author. JSON map key order is canonicalized by the
// standard encoder; the checksum deliberately excludes mutable lifecycle data.
func parserArtifactChecksum(definition map[string]any) string {
	encoded, err := json.Marshal(definition)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
func (s *Server) runParserCases(cases []assuranceRequest) map[string]any {
	passed := 0
	total := len(cases)
	results := make([]map[string]any, 0, total)
	for index, item := range cases {
		if strings.TrimSpace(item.Raw) == "" {
			results = append(results, map[string]any{"index": index, "passed": false, "errors": []string{"validation input is empty"}})
			continue
		}
		result := s.runAssurance(item)
		if result["passed"] == true {
			passed++
		}
		delete(result, "raw")
		result["index"] = index
		results = append(results, result)
	}
	status, reason := "failed", "one or more regression cases did not pass deterministic assurance"
	if total == 0 {
		status, reason = "no_cases", "add at least one validation case"
	} else if passed == total {
		status, reason = "passed", "all regression cases passed deterministic assurance"
	}
	return map[string]any{"status": status, "reason": reason, "total": total, "passed": passed, "failed": total - passed, "score": func() int {
		if total == 0 {
			return 0
		}
		return passed * 100 / total
	}(), "results": results}
}
func (s *Server) handleAssuranceParsers(w http.ResponseWriter, r *http.Request) {
	perm := middleware.PermissionReadParsers
	if r.Method != http.MethodGet {
		perm = middleware.PermissionManageParsers
	}
	p := s.authorize(w, r, perm)
	if p == nil {
		return
	}
	if r.Method == http.MethodGet {
		// Parser state is projected from the same validated runtime contract used
		// by the catalog and fleet bundle. Reading raw storage documents here can
		// otherwise leave Assurance showing a draft/active record while Parser
		// Management correctly reports the same parser as published or blocked.
		items, err := listParserDocuments(r, s.metadata)
		if err != nil {
			jsonError(w, "parser catalog unavailable", 503)
			return
		}
		jsonOK(w, map[string]any{"parsers": items, "count": len(items)})
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "GET or POST required", 405)
		return
	}
	var input assuranceParserInput
	if decodeJSON(r, &input, 2<<20) != nil || strings.TrimSpace(input.Name) == "" || len(input.Cases) == 0 || len(input.Cases) > 100 {
		jsonError(w, "name and 1..100 validation cases are required", 400)
		return
	}
	if input.Version == "" {
		input.Version = "0.1.0"
	}
	if !semverPattern.MatchString(input.Version) || input.Parser == nil {
		jsonError(w, "semantic version and declarative parser are required", 400)
		return
	}
	if err := unsafeDeclarative(input.Parser); err != nil {
		jsonError(w, "unsafe declarative parser", 400)
		return
	}
	sourceEvidenceID := safeID(input.SourceQuarantineRecordID)
	if input.SourceQuarantineRecordID != "" && sourceEvidenceID == "" {
		jsonError(w, "source quarantine record ID is invalid", 400)
		return
	}
	id := safeID(input.ID)
	if id == "" {
		id = safeID(input.Name)
	}
	if id == "" {
		jsonError(w, "parser id is invalid", 400)
		return
	}
	input.Parser["id"] = id
	if _, ok := input.Parser["version"]; !ok {
		input.Parser["version"] = 1
	}
	validation := validateParserCandidate(input.Parser, input.Cases)
	payload, _ := json.Marshal(map[string]any{"id": id, "name": input.Name, "version": input.Version, "parser": input.Parser, "artifact_checksum": parserArtifactChecksum(input.Parser), "cases": input.Cases, "validation": validation, "validation_passed": validation["failed"] == 0 && validation["total"] == len(input.Cases), "review_state": "draft", "activated": false, "source_quarantine_record_id": sourceEvidenceID})
	saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: id, State: "draft", Payload: payload, CreatedBy: p.Username})
	if err != nil {
		jsonError(w, "parser draft persistence failed", 500)
		return
	}
	s.auditAction(p, "assurance.parser_draft", "parser-packs/"+id, "success", nil)
	jsonCreated(w, map[string]any{"id": id, "revision": saved.Revision, "state": "draft", "validation": validation})
}

func (s *Server) validateParserEvidenceLink(w http.ResponseWriter, r *http.Request, recordID string) bool {
	if recordID == "" {
		return true
	}
	if s.authorize(w, r, middleware.PermissionReadLogs) == nil {
		return false
	}
	if _, _, err := s.getQuarantineRecord(r.Context(), recordID); err != nil {
		jsonError(w, "linked quarantine record not found", http.StatusNotFound)
		return false
	}
	return true
}

func (s *Server) handleAssuranceParser(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/assurance/parsers/"), "/")
	parts := strings.Split(path, "/")
	id := safeID(parts[0])
	if id == "" {
		jsonError(w, "parser id required", 400)
		return
	}
	if len(parts) < 2 {
		permission := middleware.PermissionManageParsers
		if r.Method == http.MethodGet {
			permission = middleware.PermissionReadParsers
		}
		p := s.authorize(w, r, permission)
		if p == nil {
			return
		}
		doc, err := s.metadata.Get(r.Context(), "parser-packs", id, 0)
		if err != nil {
			jsonError(w, "parser draft not found", 404)
			return
		}
		switch r.Method {
		case http.MethodGet:
			jsonOK(w, documentView(doc))
		case http.MethodPut:
			if doc.State != "draft" && doc.State != "rejected" {
				jsonError(w, "only draft or rejected parsers may be edited", 409)
				return
			}
			var input assuranceParserInput
			var previous map[string]any
			_ = json.Unmarshal(doc.Payload, &previous)
			if decodeJSON(r, &input, 2<<20) != nil || strings.TrimSpace(input.Name) == "" || len(input.Cases) == 0 || len(input.Cases) > 100 {
				jsonError(w, "name and 1..100 validation cases are required", 400)
				return
			}
			if input.Version == "" {
				input.Version = "0.1.0"
			}
			if !semverPattern.MatchString(input.Version) || input.Parser == nil || unsafeDeclarative(input.Parser) != nil {
				jsonError(w, "safe declarative parser and semantic version are required", 400)
				return
			}
			sourceEvidenceID := safeID(input.SourceQuarantineRecordID)
			if input.SourceQuarantineRecordID != "" && sourceEvidenceID == "" {
				jsonError(w, "source quarantine record ID is invalid", 400)
				return
			}
			if sourceEvidenceID == "" {
				sourceEvidenceID, _ = previous["source_quarantine_record_id"].(string)
			}
			input.Parser["id"] = id
			if _, ok := input.Parser["version"]; !ok {
				input.Parser["version"] = 1
			}
			validation := validateParserCandidate(input.Parser, input.Cases)
			if !s.validateParserEvidenceLink(w, r, sourceEvidenceID) {
				return
			}
			payload, _ := json.Marshal(map[string]any{"id": id, "name": input.Name, "version": input.Version, "parser": input.Parser, "artifact_checksum": parserArtifactChecksum(input.Parser), "cases": input.Cases, "validation": validation, "validation_passed": validation["failed"] == 0 && validation["total"] == len(input.Cases), "review_state": "draft", "activated": false, "source_quarantine_record_id": sourceEvidenceID})
			saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: id, State: "draft", Payload: payload, CreatedBy: doc.CreatedBy})
			if err != nil {
				jsonError(w, "parser draft update failed", 500)
				return
			}
			s.auditAction(p, "assurance.parser_update", "parser-packs/"+id, "success", nil)
			jsonOK(w, map[string]any{"id": id, "revision": saved.Revision, "state": "draft", "validation": validation})
		case http.MethodDelete:
			if doc.State != "draft" && doc.State != "rejected" {
				jsonError(w, "only draft or rejected parsers may be deleted", 409)
				return
			}
			if err := s.metadata.Delete(r.Context(), "parser-packs", id, false, p.Username); err != nil {
				jsonError(w, "parser draft deletion failed", 500)
				return
			}
			s.auditAction(p, "assurance.parser_delete", "parser-packs/"+id, "success", nil)
			jsonOK(w, map[string]string{"status": "deleted"})
		default:
			w.Header().Set("Allow", "GET, PUT, DELETE")
			jsonError(w, "GET, PUT or DELETE required", 405)
		}
		return
	}
	action := parts[1]
	permission := middleware.PermissionManageParsers
	if action == "approve" || action == "reject" {
		permission = middleware.PermissionReviewParsers
	} else if action == "publish" || action == "rollback" || action == "offboard" {
		permission = middleware.PermissionTriggerRollout
	}
	p := s.authorize(w, r, permission)
	if p == nil {
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", 405)
		return
	}
	doc, err := s.metadata.Get(r.Context(), "parser-packs", id, 0)
	if err != nil {
		jsonError(w, "parser draft not found", 404)
		return
	}
	var body map[string]any
	_ = json.Unmarshal(doc.Payload, &body)
	var reviewRequest struct {
		Reason string `json:"reason"`
	}
	var approval approvalRequest
	if action == "reject" {
		if decodeJSON(r, &reviewRequest, 4<<10) != nil || strings.TrimSpace(reviewRequest.Reason) == "" || len(strings.TrimSpace(reviewRequest.Reason)) > 512 {
			jsonError(w, "rejection reason must contain 1..512 characters", 400)
			return
		}
	} else if action == "approve" {
		if decodeJSON(r, &approval, 8<<10) != nil {
			jsonError(w, "approval request is invalid", http.StatusBadRequest)
			return
		}
	}
	approvalMode, approvalAudit := "", map[string]string(nil)
	switch action {
	case "validate":
		var cases []assuranceRequest
		if data, ok := body["cases"]; ok {
			encoded, _ := json.Marshal(data)
			_ = json.Unmarshal(encoded, &cases)
		}
		definition, _ := body["parser"].(map[string]any)
		if definition == nil {
			jsonError(w, "parser draft has no declarative parser", 409)
			return
		}
		if len(cases) == 0 && body["generated_by"] == "napi-bounded-v1" {
			jsonError(w, "NAPI validation evidence is immutable; rerun synthesis with separate discovery and held-out samples", 409)
			return
		}
		validation := validateParserCandidate(definition, cases)
		body["validation"] = validation
		body["validation_passed"] = validation["failed"] == 0 && validation["total"] == len(cases)
	case "submit":
		if doc.State != "draft" && doc.State != "rejected" {
			jsonError(w, "parser is not in an editable state", 409)
			return
		}
		if body["validation_passed"] != true {
			jsonError(w, "all parser validation cases must pass before review", 409)
			return
		}
		definition, _ := body["parser"].(map[string]any)
		checksum := parserArtifactChecksum(definition)
		if checksum == "" || checksum != body["artifact_checksum"] {
			jsonError(w, "parser artifact checksum changed; edit and resubmit the draft", 409)
			return
		}
		body["submitted_checksum"] = checksum
		body["review_state"] = "in_review"
	case "cancel":
		if doc.State != "in_review" {
			jsonError(w, "only a submitted parser can be withdrawn", 409)
			return
		}
		body["review_state"] = "draft"
		body["withdrawn_by"] = p.Username
		body["withdrawn_at"] = time.Now().UTC()
	case "approve":
		if doc.State != "in_review" {
			jsonError(w, "parser is not awaiting review", 409)
			return
		}
		definition, _ := body["parser"].(map[string]any)
		checksum := parserArtifactChecksum(definition)
		if checksum == "" || checksum != body["submitted_checksum"] || checksum != body["artifact_checksum"] {
			jsonError(w, "submitted parser artifact changed; approval is invalid", 409)
			return
		}
		approvalMode, approvalAudit, err = s.approvalMode(doc, p, approval, checksum, "parser_checksum")
		if err != nil {
			jsonError(w, err.Error(), http.StatusConflict)
			return
		}
		body["review_state"] = "approved"
		body["reviewed_by"] = p.Username
		body["approved_checksum"] = checksum
		recordEmergencyApproval(body, p, approval, approvalMode)
	case "reject":
		if doc.State != "in_review" {
			jsonError(w, "parser is not awaiting review", 409)
			return
		}
		if !principalCanReviewChange(p, doc.CreatedBy) {
			jsonError(w, "creator cannot reject their own parser", 409)
			return
		}
		body["review_state"] = "rejected"
		body["reviewed_by"] = p.Username
		body["review_reason"] = strings.TrimSpace(reviewRequest.Reason)
	case "publish":
		if doc.State != "approved" {
			jsonError(w, "only approved parsers can be published", 409)
			return
		}
		definition, _ := body["parser"].(map[string]any)
		checksum := parserArtifactChecksum(definition)
		if checksum == "" || checksum != body["approved_checksum"] || checksum != body["artifact_checksum"] {
			jsonError(w, "approved parser artifact checksum no longer matches", 409)
			return
		}
		var cases []assuranceRequest
		if encoded, marshalErr := json.Marshal(body["cases"]); marshalErr == nil {
			_ = json.Unmarshal(encoded, &cases)
		}
		version, _ := body["version"].(string)
		pack, packErr := assuranceRuntimePack(id, version, doc.CreatedBy, definition, cases)
		if packErr != nil {
			jsonError(w, "parser is approved but cannot be installed by the runtime: "+packErr.Error(), 409)
			return
		}
		encodedPack, encodeErr := json.Marshal(pack)
		if encodeErr != nil {
			jsonError(w, "parser runtime pack could not be encoded", 500)
			return
		}
		if _, validationErr := familyprofile.ValidateManagedPackBytes(id, encodedPack); validationErr != nil {
			jsonError(w, "parser is approved but failed the production runtime gate: "+validationErr.Error(), 409)
			return
		}
		body["runtime_pack"] = pack
		body["review_state"] = "published"
		body["activated"] = true
		body["published_by"] = p.Username
	case "rollback", "offboard":
		if doc.State != "published" || body["activated"] != true {
			jsonError(w, "only an active published parser can be rolled back or offboarded", 409)
			return
		}
		body["review_state"] = "approved"
		body["activated"] = false
		if action == "rollback" {
			body["rolled_back_by"] = p.Username
			body["rolled_back_at"] = time.Now().UTC()
		} else {
			body["offboarded_by"] = p.Username
			body["offboarded_at"] = time.Now().UTC()
		}
	default:
		jsonError(w, "unknown parser action", 404)
		return
	}
	state := fmt.Sprint(body["review_state"])
	payload, _ := json.Marshal(body)
	saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: id, State: state, Payload: payload, CreatedBy: doc.CreatedBy})
	if err != nil {
		jsonError(w, "parser lifecycle update failed", 500)
		return
	}
	response := map[string]any{"id": id, "revision": saved.Revision, "state": state, "review_state": state, "published": state == "published"}
	if action == "publish" || action == "rollback" || action == "offboard" {
		bundle, bundleErr := s.buildActiveParserBundle(r.Context())
		if bundleErr == nil {
			_, bundleErr = s.materializeParserBundle(bundle)
		}
		var proposal metastore.Document
		if bundleErr == nil {
			proposal, bundleErr = s.proposeParserBundleRollout(r.Context(), p.Username, "Assurance parser "+action+" "+id, "Parser bundle changes require independent rollout approval before client synchronization.", "all_eligible", bundle, nil)
		}
		if bundleErr != nil {
			rollbackErr := s.restoreParserDocument(r.Context(), doc)
			detail := "parser " + action + " was reverted because the fleet rollout proposal could not be prepared: " + bundleErr.Error()
			if rollbackErr != nil {
				detail += "; automatic rollback also failed: " + rollbackErr.Error()
			}
			s.auditAction(p, "assurance.parser_"+action+"_sync_failed", "parser-packs/"+id, "failure", map[string]string{"error": detail})
			jsonError(w, detail, 503)
			return
		}
		response["bundle_hash"] = bundle.BundleHash
		response["rollout_proposal"] = documentView(proposal)
		response["approval_gate"] = "pending_approval"
	}
	s.auditAction(p, "assurance.parser_"+action, "parser-packs/"+id, "success", approvalAudit)
	jsonOK(w, response)
}
func (s *Server) recordAssuranceHistory(result map[string]any, actor string) {
	id := fmt.Sprintf("run-%d", time.Now().UnixNano())
	parser, _ := result["parser"].(map[string]any)
	coverage, _ := result["coverage"].(map[string]any)
	payload, _ := json.Marshal(map[string]any{"score": result["score"], "passed": result["passed"], "parser": parser, "coverage": coverage, "created_at": time.Now().UTC()})
	_, _ = s.metadata.Put(context.Background(), metastore.Document{Collection: "assurance-history", ID: id, State: "completed", Payload: payload, CreatedBy: actor})
}
func (s *Server) handleAssuranceHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "GET required", 405)
		return
	}
	docs, err := s.metadata.List(r.Context(), "assurance-history", metastore.Query{Limit: 100})
	if err != nil {
		jsonError(w, "assurance history unavailable", 503)
		return
	}
	events := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		var payload map[string]any
		_ = json.Unmarshal(doc.Payload, &payload)
		events = append(events, map[string]any{"id": doc.ID, "state": doc.State, "created_by": doc.CreatedBy, "created_at": doc.CreatedAt, "result": payload})
	}
	sort.Slice(events, func(i, j int) bool { return fmt.Sprint(events[i]["created_at"]) > fmt.Sprint(events[j]["created_at"]) })
	jsonOK(w, map[string]any{"events": events, "count": len(events)})
}

// Test plans are declarative metadata only. Every write creates a store-managed
// immutable revision; callers may read older revisions with ?revision=N.
func (s *Server) handleAssurancePlans(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.authorize(w, r, middleware.PermissionReadParsers) == nil {
			return
		}
		docs, err := s.metadata.List(r.Context(), "assurance-plans", metastore.Query{Limit: 100})
		if err != nil {
			jsonError(w, "Assurance plans unavailable", http.StatusServiceUnavailable)
			return
		}
		plans := make([]map[string]any, 0, len(docs))
		for _, doc := range docs {
			plans = append(plans, documentView(doc))
		}
		jsonOK(w, map[string]any{"plans": plans, "count": len(plans)})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		jsonError(w, "GET or POST required", http.StatusMethodNotAllowed)
		return
	}
	principal := s.authorize(w, r, middleware.PermissionManageParsers)
	if principal == nil {
		return
	}
	s.saveAssurancePlan(w, r, principal, "")
}

func (s *Server) saveAssurancePlan(w http.ResponseWriter, r *http.Request, principal *middleware.SecurityContext, requiredID string) {
	var input struct {
		ID      string         `json:"id,omitempty"`
		Name    string         `json:"name"`
		Version string         `json:"version"`
		Plan    map[string]any `json:"plan"`
	}
	if decodeJSON(r, &input, 1<<20) != nil || strings.TrimSpace(input.Name) == "" || len(input.Name) > 128 || input.Plan == nil {
		jsonError(w, "name and declarative plan are required", http.StatusBadRequest)
		return
	}
	if input.Version == "" {
		input.Version = "1.0.0"
	}
	if !semverPattern.MatchString(input.Version) || unsafeDeclarative(input.Plan) != nil {
		jsonError(w, "valid semantic version and safe declarative plan are required", http.StatusBadRequest)
		return
	}
	if err := validateAssurancePlan(input.Plan); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := safeID(input.ID)
	if requiredID != "" {
		if input.ID != "" && input.ID != requiredID {
			jsonError(w, "body plan id must match resource id", http.StatusBadRequest)
			return
		}
		id = requiredID
	}
	if input.ID != "" && id != input.ID {
		jsonError(w, "plan id must use lowercase letters, digits, hyphens or underscores", http.StatusBadRequest)
		return
	}
	if id == "" {
		id = safeID(input.Name)
	}
	if id == "" {
		jsonError(w, "plan id is invalid", http.StatusBadRequest)
		return
	}
	payload, _ := json.Marshal(map[string]any{"schema_version": 1, "id": id, "name": strings.TrimSpace(input.Name), "version": input.Version, "plan": input.Plan})
	saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "assurance-plans", ID: id, State: "active", Payload: payload, CreatedBy: principal.Username})
	if err != nil {
		jsonError(w, "Assurance plan persistence failed", http.StatusInternalServerError)
		return
	}
	s.auditAction(principal, "assurance.plan_save", "assurance-plans/"+id, "success", map[string]string{"revision": fmt.Sprint(saved.Revision), "version": input.Version})
	jsonCreated(w, map[string]any{"id": id, "revision": saved.Revision, "version": input.Version, "state": "active", "sha256": saved.Hash})
}

func validateAssurancePlan(plan map[string]any) error {
	allowed := map[string]bool{"objective": true, "environment": true, "dataset": true, "target_version": true, "parser_versions": true, "required_features": true, "scenarios": true, "expected_results": true, "pass_criteria": true, "timeouts": true, "resource_limits": true, "cleanup": true, "evidence": true, "approvals": true, "dependencies": true}
	for key := range plan {
		if !allowed[key] {
			return fmt.Errorf("unsupported plan field %q", key)
		}
	}
	for _, key := range []string{"objective", "environment", "scenarios", "pass_criteria"} {
		if _, ok := plan[key]; !ok {
			return fmt.Errorf("plan field %q is required", key)
		}
	}
	if value, ok := plan["objective"].(string); !ok || strings.TrimSpace(value) == "" || len(value) > 2048 {
		return errors.New("objective must be a nonempty string no longer than 2048 bytes")
	}
	if value, ok := plan["environment"].(string); !ok || strings.TrimSpace(value) == "" || len(value) > 128 {
		return errors.New("environment must be a nonempty string no longer than 128 bytes")
	}
	for _, key := range []string{"scenarios", "pass_criteria"} {
		items, ok := plan[key].([]any)
		if !ok || len(items) == 0 || len(items) > 100 {
			return fmt.Errorf("%s must contain 1..100 entries", key)
		}
		for _, item := range items {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" || len(text) > 256 {
				return fmt.Errorf("%s entries must be nonempty strings no longer than 256 bytes", key)
			}
		}
	}
	if value, ok := plan["required_features"]; ok {
		items, ok := value.([]any)
		if !ok || len(items) > 100 {
			return errors.New("required_features must be an array of at most 100 strings")
		}
		for _, item := range items {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" || len(text) > 128 {
				return errors.New("required_features entries must be nonempty strings no longer than 128 bytes")
			}
		}
	}
	return nil
}

func (s *Server) handleAssurancePlan(w http.ResponseWriter, r *http.Request) {
	id := safeID(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/assurance/plans/"), "/"))
	if id == "" {
		jsonError(w, "plan id required", http.StatusBadRequest)
		return
	}
	if r.Method == http.MethodGet {
		if s.authorize(w, r, middleware.PermissionReadParsers) == nil {
			return
		}
		revision := 0
		if raw := r.URL.Query().Get("revision"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 1 {
				jsonError(w, "revision must be a positive integer", http.StatusBadRequest)
				return
			}
			revision = parsed
		}
		doc, err := s.metadata.Get(r.Context(), "assurance-plans", id, revision)
		if err != nil {
			jsonError(w, "Assurance plan not found", http.StatusNotFound)
			return
		}
		jsonOK(w, documentView(doc))
		return
	}
	if r.Method == http.MethodDelete {
		principal := s.authorize(w, r, middleware.PermissionManageParsers)
		if principal == nil {
			return
		}
		if err := s.metadata.Delete(r.Context(), "assurance-plans", id, false, principal.Username); err != nil {
			jsonError(w, "Assurance plan deletion failed", http.StatusInternalServerError)
			return
		}
		s.auditAction(principal, "assurance.plan_delete", "assurance-plans/"+id, "success", nil)
		jsonOK(w, map[string]string{"status": "deleted", "id": id})
		return
	}
	if r.Method == http.MethodPut {
		principal := s.authorize(w, r, middleware.PermissionManageParsers)
		if principal == nil {
			return
		}
		if _, err := s.metadata.Get(r.Context(), "assurance-plans", id, 0); err != nil {
			jsonError(w, "Assurance plan not found", http.StatusNotFound)
			return
		}
		s.saveAssurancePlan(w, r, principal, id)
		return
	}
	w.Header().Set("Allow", "GET, PUT, DELETE")
	jsonError(w, "GET, PUT or DELETE required", http.StatusMethodNotAllowed)
}
