package controlplane

import (
	"context"
	"errors"
	"nexora.local/nli-v20/internal/aiprovider"
	"regexp"
)

var aiRemovalID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

func validAIProfileRemovalID(id string) bool { return aiRemovalID.MatchString(id) }

// Removal is serialized with saves and late connection-test bookkeeping.
// A tombstone first makes a failed purge unavailable to future provider calls.
// Purge removes metadata revisions, not external environment/files/backups or
// an API key at its issuer. Already outbound requests cannot be recalled.
func (s *Server) removeAIProviderCredential(ctx context.Context, id, actor string, retain bool) (map[string]any, error) {
	aiProviderConfigMu.Lock()
	defer aiProviderConfigMu.Unlock()
	cfg, _, loadErr := s.loadAIProviderByID(ctx, id)
	if err := s.metadata.Delete(ctx, "ai-provider", id, false, actor); err != nil {
		return nil, errors.New("Profile could not be disabled; removal was not completed")
	}
	if err := s.metadata.Delete(ctx, "ai-provider", id, true, actor); err != nil {
		return nil, errors.New("Profile is disabled, but stored history purge was incomplete; retry removal and verify backup policy separately")
	}
	result := map[string]any{"id": id, "credential_removed": true, "stored_history_purged": true, "profile_retained": false, "configured": false, "enabled": false, "external_key_revoked": false, "external_reference_deleted": false, "note": "Only this profile and its metadata history were changed. Revoke provider keys and remove external environment/file credentials separately when required; backups and already outbound requests are outside this operation."}
	if retain && loadErr == nil {
		cfg.Enabled = false
		cfg.APIKey = ""
		cfg.LegacyAPIKey = ""
		cfg.Credential = aiprovider.Credential{}
		cfg.Headers = nil
		cfg.CredentialRemoved = true
		if err := s.storeAIProviderLocked(ctx, cfg, actor); err != nil {
			return nil, errors.New("Stored credentials were purged, but disabled metadata could not be retained; the profile is unconfigured")
		}
		result["profile_retained"] = true
		result["status"] = "credential_removed_disabled"
	} else {
		result["status"] = "removed_unconfigured"
	}
	return result, nil
}
