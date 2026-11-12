package controlplane

import (
	"context"
	"errors"
	"net/http"

	"nexora.local/nli-v20/internal/aiprovider"
)

func (s *Server) handleAIProviderModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if s.adminOnly(w, r) == nil {
		return
	}
	var input struct {
		ID string `json:"id"`
	}
	if decodeJSON(r, &input, 1024) != nil || !validAIProfileRemovalID(input.ID) {
		jsonError(w, "A saved provider profile ID is required", http.StatusBadRequest)
		return
	}
	cfg, _, err := s.loadAIProviderByID(r.Context(), input.ID)
	if err != nil {
		jsonError(w, "The saved provider profile is unavailable", http.StatusNotFound)
		return
	}
	if s.assurance == nil || cfg.CredentialRemoved {
		jsonError(w, "The saved provider credential is unavailable", http.StatusServiceUnavailable)
		return
	}
	select {
	case aiReviewSlots <- struct{}{}:
		defer func() { <-aiReviewSlots }()
	default:
		jsonError(w, "AI is busy. Retry model discovery after the current requests finish", http.StatusTooManyRequests)
		return
	}

	profile := profileFromConfig(cfg)
	policy := providerPolicy()
	resolver := aiprovider.SecretResolver{AEAD: s.assurance.providerAEAD, SecretRoot: policy.SecretRoot}
	if cfg.APIKey != "" {
		profile.Credential = aiprovider.Credential{Source: aiprovider.CredentialEnvironment, Reference: "NLI_AI_TRANSIENT_CREDENTIAL"}
		resolver.Getenv = func(string) string { return cfg.APIKey }
	}
	catalog, err := aiprovider.DiscoverModels(r.Context(), profile, resolver, policy)
	if err != nil {
		switch {
		case errors.Is(err, aiprovider.ErrModelDiscoveryUnsupported):
			jsonError(w, "This provider does not expose a supported model list. Enter a model ID from its catalog.", http.StatusUnprocessableEntity)
		case errors.Is(err, context.DeadlineExceeded):
			jsonError(w, "The provider model catalog timed out", http.StatusGatewayTimeout)
		default:
			jsonError(w, "The provider model catalog could not be loaded. Check the saved credentials and endpoint.", http.StatusBadGateway)
		}
		return
	}
	s.recordAIAuthoring("provider_models", "")
	jsonOK(w, map[string]any{"profile_id": cfg.ID, "models": catalog.Models, "truncated": catalog.Truncated, "capabilities_verified": false})
}
