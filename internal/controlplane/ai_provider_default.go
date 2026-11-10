package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"nexora.local/nli-v20/internal/aiprovider"
	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

const aiProviderRoutingCollection = "ai-provider-routing"
const aiProviderRoutingID = "controller"

type aiProviderRouting struct {
	DefaultProfileID string   `json:"default_profile_id"`
	ProfileOrder     []string `json:"profile_order"`
}

func (s *Server) loadAIProviderRouting(ctx context.Context) (aiProviderRouting, error) {
	routing := aiProviderRouting{ProfileOrder: []string{}}
	doc, err := s.metadata.Get(ctx, aiProviderRoutingCollection, aiProviderRoutingID, 0)
	if errors.Is(err, os.ErrNotExist) {
		return routing, nil
	}
	if err != nil {
		return routing, err
	}
	if err := json.Unmarshal(doc.Payload, &routing); err != nil {
		return aiProviderRouting{}, err
	}
	if routing.DefaultProfileID != "" && !validAIProfileRemovalID(routing.DefaultProfileID) {
		return aiProviderRouting{}, errors.New("invalid stored default AI provider profile")
	}
	if !validAIProfileOrder(routing.ProfileOrder) {
		return aiProviderRouting{}, errors.New("invalid stored AI provider order")
	}
	return routing, nil
}

func validAIProfileOrder(ids []string) bool {
	if len(ids) > 16 {
		return false
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !validAIProfileRemovalID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func (s *Server) handleAIProviderRouting(w http.ResponseWriter, r *http.Request, principal *middleware.SecurityContext, action, id string) {
	if (action == "set-default" && !validAIProfileRemovalID(id)) || (action != "set-default" && id != "") {
		jsonError(w, "Set a default with an exact profile ID or clear the default without an ID", http.StatusBadRequest)
		return
	}
	var orderRequest struct {
		ProfileOrder []string `json:"profile_order"`
	}
	if action == "set-order" && (decodeJSON(r, &orderRequest, 4096) != nil || orderRequest.ProfileOrder == nil || !validAIProfileOrder(orderRequest.ProfileOrder)) {
		jsonError(w, "Provide profile_order with up to 16 unique valid profile IDs", http.StatusBadRequest)
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
	routing, err := s.loadAIProviderRouting(r.Context())
	if err != nil {
		jsonError(w, "AI provider routing could not be read", http.StatusServiceUnavailable)
		return
	}
	if action == "set-order" {
		routing.ProfileOrder = orderRequest.ProfileOrder
	} else {
		routing.DefaultProfileID = id
	}
	payload, _ := json.Marshal(routing)
	_, err = s.metadata.Put(r.Context(), metastore.Document{
		Collection: aiProviderRoutingCollection, ID: aiProviderRoutingID,
		State: "configured", Payload: payload, CreatedBy: principal.Username,
	})
	if err != nil {
		jsonError(w, "AI provider routing could not be saved", http.StatusServiceUnavailable)
		return
	}
	s.auditAction(principal, "ai_provider."+action, aiProviderRoutingCollection+"/"+aiProviderRoutingID, "success", map[string]string{"profile_id": id})
	s.publishAIProviderStatus("routing_changed", "", "")
	jsonOK(w, routing)
}

func orderedAIProviderRoute(profiles []aiprovider.Profile, order []string) []aiprovider.Profile {
	route := aiprovider.Route(profiles, "")
	if len(order) == 0 {
		return route
	}
	byID := make(map[string]aiprovider.Profile, len(route))
	for _, profile := range route {
		byID[profile.ID] = profile
	}
	ordered := make([]aiprovider.Profile, 0, len(route))
	for _, id := range order {
		if profile, ok := byID[id]; ok {
			ordered = append(ordered, profile)
			delete(byID, id)
		}
	}
	for _, profile := range route {
		if _, ok := byID[profile.ID]; ok {
			ordered = append(ordered, profile)
		}
	}
	return ordered
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
