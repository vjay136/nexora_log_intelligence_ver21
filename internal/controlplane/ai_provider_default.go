package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

const aiProviderRoutingCollection = "ai-provider-routing"
const aiProviderRoutingID = "controller"

type aiProviderRouting struct {
	DefaultProfileID string `json:"default_profile_id"`
}

func (s *Server) aiProviderDefaultID(ctx context.Context) (string, error) {
	doc, err := s.metadata.Get(ctx, aiProviderRoutingCollection, aiProviderRoutingID, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var routing aiProviderRouting
	if err := json.Unmarshal(doc.Payload, &routing); err != nil {
		return "", err
	}
	if routing.DefaultProfileID != "" && !validAIProfileRemovalID(routing.DefaultProfileID) {
		return "", errors.New("invalid stored default AI provider profile")
	}
	return routing.DefaultProfileID, nil
}

func (s *Server) handleAIProviderDefault(w http.ResponseWriter, r *http.Request, principal *middleware.SecurityContext, action, id string) {
	if (action == "set-default" && !validAIProfileRemovalID(id)) || (action == "clear-default" && id != "") {
		jsonError(w, "Set a default with an exact profile ID or clear the default without an ID", http.StatusBadRequest)
		return
	}
	aiProviderConfigMu.Lock()
	defer aiProviderConfigMu.Unlock()
	if id != "" {
		cfg, _, err := s.loadAIProviderByID(r.Context(), id)
		if err != nil || !cfg.Enabled {
			jsonError(w, "Select an enabled, readable AI provider profile", http.StatusConflict)
			return
		}
	}
	payload, _ := json.Marshal(aiProviderRouting{DefaultProfileID: id})
	_, err := s.metadata.Put(r.Context(), metastore.Document{
		Collection: aiProviderRoutingCollection, ID: aiProviderRoutingID,
		State: "configured", Payload: payload, CreatedBy: principal.Username,
	})
	if err != nil {
		jsonError(w, "Default AI provider selection could not be saved", http.StatusServiceUnavailable)
		return
	}
	s.auditAction(principal, "ai_provider."+action, aiProviderRoutingCollection+"/"+aiProviderRoutingID, "success", map[string]string{"profile_id": id})
	s.publishAIProviderStatus("routing_changed", "", "")
	jsonOK(w, map[string]any{"default_profile_id": id})
}

func (s *Server) aiProviderRouteConfigs(ctx context.Context, primary aiProviderConfig) ([]aiProviderConfig, error) {
	if primary.Exclusive {
		cfg, _, err := s.loadAIProviderByID(ctx, primary.ID)
		if err != nil || !cfg.Enabled {
			return nil, errors.New("default AI provider profile is unavailable")
		}
		return []aiProviderConfig{cfg}, nil
	}
	return s.loadAIProviders(ctx)
}
