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
	Provider          string                `json:"provider"`
	BaseURL           string                `json:"base_url"`
	Model             string                `json:"model"`
	APIKey            string                `json:"-"`
	LegacyAPIKey      string                `json:"api_key,omitempty"`
	ID                string                `json:"id,omitempty"`
	Name              string                `json:"name,omitempty"`
	Enabled           bool                  `json:"enabled"`
	Priority          int                   `json:"priority,omitempty"`
	FallbackGroup     string                `json:"fallback_group,omitempty"`
	Credential        aiprovider.Credential `json:"credential,omitempty"`
	Headers           map[string]string     `json:"headers,omitempty"`
	TimeoutSeconds    int                   `json:"timeout_seconds,omitempty"`
	MaxResponseBytes  int64                 `json:"max_response_bytes,omitempty"`
	CredentialRemoved bool                  `json:"credential_removed,omitempty"`
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
	aiProviderConfigMu.Lock()
	defer aiProviderConfigMu.Unlock()
	return s.storeAIProviderLocked(ctx, cfg, actor)
}
func (s *Server) storeAIProviderLocked(ctx context.Context, cfg aiProviderConfig, actor string) error {
	if s.assurance.providerAEAD == nil {
		return errors.New("Assurance encryption key is not configured")
	}
	cfg = normalizeProvider(cfg)
	if _, err := s.metadata.Get(ctx, "ai-provider", cfg.ID, 0); errors.Is(err, os.ErrNotExist) {
		docs, listErr := s.metadata.List(ctx, "ai-provider", metastore.Query{Limit: 17})
		if listErr != nil {
			return listErr
		}
		if len(docs) >= 16 {
			return errors.New("AI provider catalog is limited to 16 profiles")
		}
	} else if err != nil {
		return err
	}
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
	_, err = s.metadata.Put(ctx, metastore.Document{Collection: "ai-provider", ID: cfg.ID, State: "configured", Payload: payload, CreatedBy: actor})
	return err
}
func (s *Server) loadAIProviderByID(ctx context.Context, id string) (aiProviderConfig, sealedAIProvider, error) {
	doc, err := s.metadata.Get(ctx, "ai-provider", id, 0)
	if err != nil {
		return aiProviderConfig{}, sealedAIProvider{}, err
	}
	return s.openAIProvider(doc)
}
func (s *Server) openAIProvider(doc metastore.Document) (aiProviderConfig, sealedAIProvider, error) {
	if s.assurance.providerAEAD == nil {
		return aiProviderConfig{}, sealedAIProvider{}, errors.New("Assurance encryption key is not configured")
	}
	var sealed sealedAIProvider
	if err := json.Unmarshal(doc.Payload, &sealed); err != nil {
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
	if cfg.ID != doc.ID {
		return aiProviderConfig{}, sealedAIProvider{}, errors.New("provider profile identity mismatch")
	}
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
	return aiprovider.Normalize(aiprovider.Profile{ID: cfg.ID, Name: cfg.Name, Kind: providerKind(cfg.Provider), BaseURL: cfg.BaseURL, Model: cfg.Model, Enabled: cfg.Enabled, Priority: cfg.Priority, FallbackGroup: cfg.FallbackGroup, TimeoutSeconds: cfg.TimeoutSeconds, MaxResponseBytes: cfg.MaxResponseBytes, Headers: cfg.Headers, Credential: cfg.Credential})
}

func validateProvider(cfg aiProviderConfig) error {
	policy := providerPolicy()
	policy.AllowLoopback = true // retained for isolated protocol-adapter tests
	return validateProviderWithPolicy(cfg, policy)
}

func validateProviderWithPolicy(cfg aiProviderConfig, policy aiprovider.ValidationPolicy) error {
	cfg = normalizeProvider(cfg)
	profile := profileFromConfig(cfg)
	if cfg.CredentialRemoved && !cfg.Enabled && cfg.APIKey == "" && cfg.LegacyAPIKey == "" && cfg.Credential == (aiprovider.Credential{}) && len(cfg.Headers) == 0 {
		return aiprovider.ValidateMetadata(profile, policy)
	}
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
	status := "configured_unverified"
	if !cfg.Enabled {
		status = "disabled"
	} else if sealed.LastTestAt != nil {
		status = "last_test_succeeded"
	}
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
		"enabled": view.Enabled, "status": status, "credential_removed": cfg.CredentialRemoved, "key_last4": view.CredentialLast4,
		"credential_source": view.CredentialSource, "credential_ref": view.CredentialRef,
		"priority": view.Priority, "fallback_group": view.FallbackGroup, "last_test_at": sealed.LastTestAt, "connection_verified": sealed.LastTestAt != nil, "timeout_seconds": view.TimeoutSeconds, "max_response_bytes": view.MaxResponseBytes,
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
	case http.MethodDelete, http.MethodPost:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if !validAIProfileRemovalID(id) {
			jsonError(w, "An exact valid profile id is required", 400)
			return
		}
		clear := r.Method == http.MethodPost
		if clear && r.URL.Query().Get("action") != "clear-credential" {
			jsonError(w, "Supported POST action is clear-credential", 405)
			return
		}
		result, err := s.removeAIProviderCredential(r.Context(), id, p.Username, clear)
		action := "ai_provider.delete"
		if clear {
			action = "ai_provider.clear_credential"
		}
		if err != nil {
			s.auditAction(p, action, "ai-provider/"+id, "failure", nil)
			jsonError(w, err.Error(), 503)
			return
		}
		s.auditAction(p, action, "ai-provider/"+id, "success", nil)
		s.publishAIProviderStatus("credential_removed", "", "")
		jsonOK(w, result)
	case http.MethodGet:
		jsonOK(w, s.aiProviderCatalog(r.Context(), strings.TrimSpace(r.URL.Query().Get("id"))))
	case http.MethodPut:
		aiProviderConfigMu.Lock()
		defer aiProviderConfigMu.Unlock()
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
			TimeoutSeconds   int               `json:"timeout_seconds"`
			MaxResponseBytes int64             `json:"max_response_bytes"`
			Enabled          *bool             `json:"enabled"`
		}
		if decodeJSON(r, &request, 16<<10) != nil {
			jsonError(w, "invalid provider configuration", 400)
			return
		}

		if request.ID != "" && (!validAIProfileRemovalID(request.ID) || request.ID != strings.ToLower(request.ID)) {
			jsonError(w, "New profile IDs must be lowercase letters, digits, underscore or hyphen", 400)
			return
		}

		kind := request.Kind
		if kind == "" {
			kind = request.Provider
		}
		cfg := aiProviderConfig{Provider: kind, ID: request.ID, Name: request.Name, BaseURL: request.BaseURL, Model: request.Model, APIKey: request.APIKey, Enabled: true, Priority: request.Priority, FallbackGroup: request.FallbackGroup, Headers: request.Headers, TimeoutSeconds: request.TimeoutSeconds, MaxResponseBytes: request.MaxResponseBytes}
		if cfg.BaseURL == "" {
			cfg.BaseURL = request.Endpoint
		}
		cfg = normalizeProvider(cfg)
		if request.Enabled != nil {
			cfg.Enabled = *request.Enabled
		}
		previous, _, previousErr := s.loadAIProviderByID(r.Context(), cfg.ID)
		if previousErr == nil {
			if request.Headers == nil {
				cfg.Headers = previous.Headers
			}
			if request.TimeoutSeconds == 0 {
				cfg.TimeoutSeconds = previous.TimeoutSeconds
			}
			if request.MaxResponseBytes == 0 {
				cfg.MaxResponseBytes = previous.MaxResponseBytes
			}
			if request.Enabled == nil {
				cfg.Enabled = previous.Enabled
			}
		}
		if request.CredentialSource == "" {
			request.CredentialSource = string(aiprovider.CredentialEncryptedInline)
		}
		if cfg.APIKey != "" && request.CredentialSource != string(aiprovider.CredentialEncryptedInline) {
			jsonError(w, "Inline key input requires encrypted_inline mode; use only the selected protected credential reference otherwise", 400)
			return
		}
		if cfg.APIKey == "" && request.CredentialSource == string(aiprovider.CredentialEncryptedInline) {
			if previousErr == nil {
				if previous.Credential.Source == aiprovider.CredentialEncryptedInline {
					cfg.Credential = previous.Credential
				} else if previous.APIKey != "" {
					// Transparently migrate the legacy encrypted record to the
					// provider-neutral inline credential envelope on the next save.
					cfg.APIKey = previous.APIKey
				}
			}
		} else if request.CredentialSource == string(aiprovider.CredentialEnvironment) {
			reference := request.CredentialRef
			if reference == "" && previousErr == nil && previous.Credential.Source == aiprovider.CredentialEnvironment {
				reference = previous.Credential.Reference
			}
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: reference}
		} else if request.CredentialSource == string(aiprovider.CredentialFile) {
			reference := request.CredentialRef
			if reference == "" && previousErr == nil && previous.Credential.Source == aiprovider.CredentialFile {
				reference = previous.Credential.Reference
			}
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialFile, Reference: reference}
		} else if request.CredentialSource == string(aiprovider.CredentialNone) {
			cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialNone}
		}
		if !cfg.Enabled && cfg.APIKey == "" && cfg.Credential == (aiprovider.Credential{}) && previousErr == nil && previous.CredentialRemoved {
			cfg.CredentialRemoved = true
		}
		if err := validateProviderWithPolicy(cfg, providerPolicy()); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if err := s.storeAIProviderLocked(r.Context(), cfg, p.Username); err != nil {
			jsonError(w, "provider configuration could not be encrypted or stored", 503)
			return
		}
		s.auditAction(p, "ai_provider.configure", "ai-provider/"+cfg.ID, "success", map[string]string{"provider": cfg.Provider, "model": cfg.Model})
		s.publishAIProviderStatus("configured", cfg.Provider, cfg.Model)
		result := s.aiProviderCatalog(r.Context(), cfg.ID)
		result["saved"] = true
		jsonOK(w, result)
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
	profile := profileFromConfig(cfg)
	options := aiprovider.RequestOptions{SystemPrompt: aiReviewSystemPrompt, MaxTokens: 2048}
	if cfg.APIKey != "" {
		profile.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: "NLI_AI_TRANSIENT_CREDENTIAL"}
		return aiprovider.GenerateWithCredential(ctx, profile, cfg.APIKey, policy, prompt, options)
	}
	return aiprovider.Generate(ctx, profile, aiprovider.SecretResolver{AEAD: s.assurance.providerAEAD, SecretRoot: policy.SecretRoot}, policy, prompt, options)
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
	select {
	case aiReviewSlots <- struct{}{}:
		defer func() { <-aiReviewSlots }()
	default:
		jsonError(w, "AI test is busy; retry after the current bounded requests finish", 429)
		return
	}
	var input struct {
		TimeoutSeconds   int    `json:"timeout_seconds"`
		MaxResponseBytes int64  `json:"max_response_bytes"`
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
		Stream           bool   `json:"stream"`
		ImageURL         string `json:"image_url"`
		TestMaxTokens    int    `json:"test_max_tokens"`
	}
	if r.ContentLength > 0 && decodeJSON(r, &input, 16<<10) != nil {
		jsonError(w, "invalid provider test request", 400)
		return
	}
	cfg, _, loadErr := s.loadAIProvider(r.Context())
	if input.ID != "" {
		cfg, _, loadErr = s.loadAIProviderByID(r.Context(), input.ID)
	}
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
	if input.TimeoutSeconds != 0 {
		cfg.TimeoutSeconds = input.TimeoutSeconds
	}
	if input.MaxResponseBytes != 0 {
		cfg.MaxResponseBytes = input.MaxResponseBytes
	}
	cfg = normalizeProvider(cfg)
	if input.APIKey != "" && input.CredentialSource != "" && input.CredentialSource != string(aiprovider.CredentialEncryptedInline) {
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "Inline key input requires encrypted_inline mode; clear it when selecting a controller reference.", "error_code": "invalid_credential_mode"})
		return
	}
	if input.APIKey != "" {
		credential, sealErr := aiprovider.SealInline(s.assurance.providerAEAD, cfg.ID, input.APIKey)
		if sealErr != nil {
			jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "Provider credential is invalid; enter an actual key without Bearer/$ENV placeholders, or use an environment reference."})
			return
		}
		cfg.Credential = credential
	} else if input.CredentialSource == string(aiprovider.CredentialEnvironment) {
		reference := input.CredentialRef
		if reference == "" && cfg.Credential.Source == aiprovider.CredentialEnvironment {
			reference = cfg.Credential.Reference
		}
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: reference}
	} else if input.CredentialSource == string(aiprovider.CredentialFile) {
		reference := input.CredentialRef
		if reference == "" && cfg.Credential.Source == aiprovider.CredentialFile {
			reference = cfg.Credential.Reference
		}
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialFile, Reference: reference}
	} else if input.CredentialSource == string(aiprovider.CredentialNone) {
		cfg.Credential = aiprovider.Credential{Source: aiprovider.CredentialNone}
	}
	if loadErr != nil && cfg.Credential.Source == "" {
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": "AI provider is not configured"})
		return
	}
	testInfo := s.aiTestInfo(r.Context(), cfg, input.Stream, input.ImageURL, input.TestMaxTokens)
	policy := providerPolicy()
	profile := profileFromConfig(cfg)
	if err := aiprovider.Validate(profile, policy); err != nil {
		code, message := "invalid_provider_config", "Check the provider endpoint, model, profile ID, and request limits."
		if aiprovider.ValidateMetadata(profile, policy) == nil {
			code = "invalid_credential_config"
			switch profile.Credential.Source {
			case aiprovider.CredentialEnvironment:
				message = "Enter the uppercase controller environment variable name in Credential reference, for example GEMINI_API_KEY."
			case aiprovider.CredentialFile:
				message = "Enter a valid managed secret file path in Credential reference."
			case aiprovider.CredentialEncryptedInline:
				message = "Enter an API key in Encrypted inline mode or load a saved inline credential."
			default:
				message = "Choose an available credential mode for this provider."
			}
		}
		jsonOK(w, map[string]any{"connected": false, "status": "failed", "message": message, "error_code": code, "tested_config": testInfo})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 17*time.Second)
	defer cancel()
	result := (aiprovider.Tester{Policy: policy, Resolver: aiprovider.SecretResolver{AEAD: s.assurance.providerAEAD, SecretRoot: policy.SecretRoot}}).TestWithOptions(ctx, profileFromConfig(cfg), aiprovider.RequestOptions{Stream: input.Stream, ImageURL: input.ImageURL, MaxTokens: input.TestMaxTokens})
	if !result.OK {
		if result.Acknowledged && result.ErrorCode == "incomplete_output" {
			s.auditAction(p, "ai_provider.test", "ai-provider/"+cfg.ID, "acknowledged", nil)
			s.publishAIProviderStatus("acknowledged_incomplete", cfg.Provider, cfg.Model)
			jsonOK(w, map[string]any{"connected": false, "acknowledged": true, "status": "acknowledged_incomplete", "message": "Provider acknowledged the model, but the bounded test did not produce a complete usable reply. Review token budget and model settings.", "error_code": result.ErrorCode, "status_code": result.StatusCode, "tested_config": testInfo})
			return
		}
		s.auditAction(p, "ai_provider.test", "ai-provider/"+cfg.ID, "failure", nil)
		s.publishAIProviderStatus("connection_failed", cfg.Provider, cfg.Model)
		jsonOK(w, map[string]any{"connected": false, "acknowledged": result.Acknowledged, "usable_reply": false, "status": "failed", "message": aiConnectionMessage(result.ErrorCode), "error_code": result.ErrorCode, "status_code": result.StatusCode, "tested_config": testInfo})
		return
	}
	verifiedErr := s.markAIProviderTestedExact(r.Context(), cfg, p.Username)
	if testInfo["configuration"] == "saved_profile" && verifiedErr != nil {
		jsonOK(w, map[string]any{"connected": false, "acknowledged": true, "usable_reply": true, "status": "configuration_changed", "message": "The earlier request received a reply, but the saved profile changed or was removed; no current readiness was recorded.", "tested_config": testInfo})
		return
	}
	s.auditAction(p, "ai_provider.test", "ai-provider/"+cfg.ID, "success", nil)
	s.publishAIProviderStatus("connected", cfg.Provider, cfg.Model)
	jsonOK(w, map[string]any{"connected": true, "acknowledged": true, "usable_reply": true, "status": "success", "message": "Provider connection succeeded", "model": cfg.Model, "status_code": result.StatusCode, "latency_ms": result.Latency.Milliseconds(), "tested_config": testInfo})
}

func (s *Server) markProviderTested(ctx context.Context, actor string) error {
	return s.markAIProviderTested(ctx, "default", actor)
}
func (s *Server) markAIProviderTested(ctx context.Context, id, actor string) error {
	aiProviderConfigMu.Lock()
	defer aiProviderConfigMu.Unlock()
	cfg, _, err := s.loadAIProviderByID(ctx, id)
	if err != nil {
		return err
	}
	return s.markAIProviderTestedLocked(ctx, cfg, actor)
}
func (s *Server) markAIProviderTestedExact(ctx context.Context, cfg aiProviderConfig, actor string) error {
	aiProviderConfigMu.Lock()
	defer aiProviderConfigMu.Unlock()
	return s.markAIProviderTestedLocked(ctx, cfg, actor)
}
func (s *Server) markAIProviderTestedLocked(ctx context.Context, cfg aiProviderConfig, actor string) error {
	current, sealed, err := s.loadAIProviderByID(ctx, cfg.ID)
	if err != nil {
		return err
	}
	if !current.Enabled || !sameAIProfile(current, cfg) {
		return errors.New("profile changed during test")
	}
	now := time.Now().UTC()
	sealed.LastTestAt = &now
	payload, _ := json.Marshal(sealed)
	_, err = s.metadata.Put(ctx, metastore.Document{Collection: "ai-provider", ID: cfg.ID, State: "configured", Payload: payload, CreatedBy: actor})
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
	if !cfg.Enabled {
		jsonError(w, "AI assistance is disabled", 503)
		return
	}
	select {
	case aiReviewSlots <- struct{}{}:
		defer func() { <-aiReviewSlots }()
	default:
		jsonError(w, "AI review is busy; deterministic parsing remains available", 429)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 17*time.Second)
	defer cancel()
	prompt := s.aiReviewPromptWithContext(ctx, input, result)
	aiStarted := time.Now()
	aiMetricsSuccess := false
	defer func() { s.aiMetrics.record(aiMetricsSuccess, time.Since(aiStarted)) }()
	raw, usedProvider, attempts, err := s.callConfiguredAI(ctx, cfg, prompt)
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
	if err := validateAIParserSyntax(parser); err != nil {
		jsonError(w, "AI proposal is not valid NLI parser syntax; no proposal was saved. Use the schema-aware Suggest with AI mapping option or correct the native parser definition.", 422)
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
		"provider_id":             usedProvider, "provider_attempts": attempts,
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
	jsonOK(w, map[string]any{"id": saved.ID, "proposal": proposal, "suggestions": suggestions, "categories": []string{"parser", "regex", "template", "kv", "field_mapping", "projection"}, "review_state": "draft", "requires_human_approval": true, "published": false, "stored": true, "provider_id": usedProvider, "provider_attempts": attempts})
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
	ctx, done, admitted := s.beginFieldSenseAnalysis(w, r)
	if !admitted {
		return
	}
	defer done()
	var input assuranceParserInput
	if decodeJSON(r, &input, 2<<20) != nil || input.Parser == nil || len(input.Cases) == 0 || len(input.Cases) > 100 {
		jsonError(w, "declarative parser and 1..100 validation cases are required", http.StatusBadRequest)
		return
	}
	if err := unsafeDeclarative(input.Parser); err != nil {
		jsonError(w, "unsafe declarative parser", http.StatusBadRequest)
		return
	}
	validation := validateParserCandidateContext(ctx, input.Parser, input.Cases)
	if fieldSenseContextError(w, ctx) {
		return
	}
	jsonOK(w, map[string]any{
		"validation":        validation,
		"artifact_checksum": parserArtifactChecksum(input.Parser),
		"passed":            validation["failed"] == 0 && validation["total"] == len(input.Cases),
		"persisted":         false,
		"activated":         false,
	})
}

func validateParserCandidate(definition map[string]any, cases []assuranceRequest) map[string]any {
	return validateParserCandidateContext(context.Background(), definition, cases)
}

func validateParserCandidateContext(ctx context.Context, definition map[string]any, cases []assuranceRequest) map[string]any {
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
		if err := ctx.Err(); err != nil {
			return map[string]any{"status": "interrupted", "reason": err.Error(), "total": len(cases), "passed": passed, "failed": len(cases) - passed, "errors": []string{err.Error()}}
		}
		if strings.TrimSpace(item.Raw) == "" {
			outcomes = append(outcomes, map[string]any{"index": index, "passed": false, "actual": map[string]any{}, "diff": []map[string]any{}, "errors": []string{"validation input is empty"}})
			continue
		}
		actual, errors := registry.ApplyContext(ctx, &parser, item.Raw)
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
	s.parserMutationMu.Lock()
	defer s.parserMutationMu.Unlock()
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
	if familyprofile.IsBuiltinPackID(id) {
		jsonError(w, "add-on parsers require a unique ID; built-in parser identities are protected", http.StatusConflict)
		return
	}
	if _, existingErr := s.metadata.Get(r.Context(), "parser-packs", id, 0); existingErr == nil {
		jsonError(w, "parser id already exists; edit its draft or create a repair draft", http.StatusConflict)
		return
	} else if !errors.Is(existingErr, os.ErrNotExist) {
		jsonError(w, "parser draft uniqueness cannot be verified", http.StatusServiceUnavailable)
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
	if r.Method != http.MethodGet {
		s.parserMutationMu.Lock()
		defer s.parserMutationMu.Unlock()
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
			updated := map[string]any{"id": id, "name": input.Name, "version": input.Version, "parser": input.Parser, "artifact_checksum": parserArtifactChecksum(input.Parser), "cases": input.Cases, "validation": validation, "validation_passed": validation["failed"] == 0 && validation["total"] == len(input.Cases), "review_state": "draft", "activated": false, "source_quarantine_record_id": sourceEvidenceID}
			for _, key := range []string{"repair_of", "repair_of_revision", "repair_of_checksum", "repair_created_at", "repair_created_by"} {
				if value, ok := previous[key]; ok {
					updated[key] = value
				}
			}
			payload, _ := json.Marshal(updated)
			saved, err := s.metadata.Put(r.Context(), metastore.Document{Collection: "parser-packs", ID: id, State: "draft", Payload: payload, CreatedBy: doc.CreatedBy})
			if err != nil {
				jsonError(w, "parser draft update failed", 500)
				return
			}
			s.auditAction(p, "assurance.parser_update", "parser-packs/"+id, "success", nil)
			jsonOK(w, map[string]any{"id": id, "revision": saved.Revision, "state": "draft", "validation": validation})
		case http.MethodDelete:
			s.deleteParserDocument(w, r, p, doc)
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
	if action == "clone-draft" {
		s.cloneParserRepairDraft(w, r, p, doc, body)
		return
	}
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
		if repairOf, _ := body["repair_of"].(string); repairOf != "" {
			origin, originErr := s.metadata.Get(r.Context(), "parser-packs", repairOf, 0)
			var original map[string]any
			if originErr == nil {
				originErr = json.Unmarshal(origin.Payload, &original)
			}
			if originErr != nil || original["artifact_checksum"] != body["repair_of_checksum"] {
				jsonError(w, "repair source changed; create a fresh repair draft from the current parser", http.StatusConflict)
				return
			}
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
