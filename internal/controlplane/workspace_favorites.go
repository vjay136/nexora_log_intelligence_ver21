package controlplane

import (
	"net/http"
	"regexp"

	"nexora.local/nli-v20/internal/middleware"
)

var workspaceFavoriteID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func (s *Server) handleWorkspaceFavorites(w http.ResponseWriter, r *http.Request) {
	principal := middleware.GetSecurityContext(r)
	if principal == nil || principal.Username == "" {
		jsonError(w, "active account required", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		jsonOK(w, map[string]any{"favorites": s.auth.WorkspaceFavorites(principal.Username, principal.Role)})
	case http.MethodPut:
		var input struct {
			Workspace string `json:"workspace"`
			Starred   *bool  `json:"starred"`
		}
		if err := decodeJSON(r, &input, 1024); err != nil || input.Starred == nil || !workspaceFavoriteID.MatchString(input.Workspace) {
			jsonError(w, "valid workspace and starred state required", http.StatusBadRequest)
			return
		}
		favorites, err := s.auth.SetWorkspaceFavorite(principal.Username, principal.Role, input.Workspace, *input.Starred)
		if err != nil {
			jsonError(w, "unable to save workspace favorite", http.StatusConflict)
			return
		}
		jsonOK(w, map[string]any{"favorites": favorites})
	default:
		w.Header().Set("Allow", "GET, PUT")
		jsonError(w, "GET or PUT required", http.StatusMethodNotAllowed)
	}
}
