package controlplane

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/auth"
	"nexora.local/nli-v20/internal/middleware"
	metastore "nexora.local/nli-v20/internal/store"
)

// approvalRequest is shared by parser-catalog and rollout approval handlers.
// EmergencySelfApproval is never inferred from the actor's role: the operator
// must opt in for this exact request and bind the approval to the current
// revision and checksum.
type approvalRequest struct {
	Reason                string `json:"reason"`
	EmergencySelfApproval bool   `json:"emergency_self_approval"`
	ExpectedRevision      int    `json:"expected_revision"`
	ExpectedChecksum      string `json:"expected_checksum,omitempty"`
	ExpectedBundleHash    string `json:"expected_bundle_hash,omitempty"`
}

func principalHasPermission(principal *middleware.SecurityContext, permission string) bool {
	if principal == nil {
		return false
	}
	for _, granted := range principal.Permissions {
		if granted == permission {
			return true
		}
	}
	return false
}

// principalCanReviewChange applies after the route's permission check. Only
// an authenticated administrator may review a change they created; usernames
// and individual emergency permissions never grant that exception.
func principalCanReviewChange(principal *middleware.SecurityContext, createdBy string) bool {
	return principal != nil && principal.Username != "" &&
		(principal.Username != createdBy || principal.Role == auth.RoleAdmin)
}

// approvalMode preserves independent approval as the normal path. An
// administrator may also approve their own change when they provide an
// auditable reason. Emergency self-approval remains available for callers that
// want to bind the decision to an exact revision and digest, but is no longer
// the only way for an administrator to finish the governed QRW workflow.
func (s *Server) approvalMode(doc metastore.Document, principal *middleware.SecurityContext, request approvalRequest, expectedDigest, digestField string) (string, map[string]string, error) {
	if doc.CreatedBy != principal.Username {
		details := map[string]string{"approval_mode": "independent"}
		if reason := strings.TrimSpace(request.Reason); reason != "" {
			if len(reason) < 3 || len(reason) > 512 {
				return "", nil, fmt.Errorf("independent approval reason must contain 3..512 characters")
			}
			details["approval_reason"] = reason
		}
		return "independent", details, nil
	}
	if !principalCanReviewChange(principal, doc.CreatedBy) {
		return "", nil, fmt.Errorf("creator cannot approve own change; administrator approval permission is required")
	}
	reason := strings.TrimSpace(request.Reason)
	if len(reason) < 12 || len(reason) > 512 {
		return "", nil, fmt.Errorf("administrator self-approval reason must contain 12..512 characters")
	}
	if !request.EmergencySelfApproval {
		return "admin_self", map[string]string{
			"approval_mode":     "admin_self",
			"approval_reason":   reason,
			"approved_revision": fmt.Sprint(doc.Revision),
			digestField:         expectedDigest,
		}, nil
	}
	if request.ExpectedRevision != doc.Revision {
		return "", nil, fmt.Errorf("emergency self-approval revision mismatch: expected %d", doc.Revision)
	}
	providedDigest := strings.TrimSpace(request.ExpectedChecksum)
	if digestField == "bundle_hash" {
		providedDigest = strings.TrimSpace(request.ExpectedBundleHash)
	}
	if expectedDigest == "" || providedDigest != expectedDigest {
		return "", nil, fmt.Errorf("emergency self-approval %s mismatch", digestField)
	}
	details := map[string]string{
		"approval_mode":     "emergency_admin_self",
		"approval_reason":   reason,
		"approved_revision": fmt.Sprint(doc.Revision),
		digestField:         expectedDigest,
	}
	return "emergency_admin_self", details, nil
}

// optionalApprovalComment preserves the approval contract for assets and AI proposals
// that historically accepted an empty request body.
func optionalApprovalComment(r *http.Request, doc metastore.Document, principal *middleware.SecurityContext) (approvalRequest, string, error) {
	var request approvalRequest
	if err := decodeJSON(r, &request, 8<<10); err != nil && !errors.Is(err, io.EOF) {
		return request, "", errors.New("approval request is invalid")
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if length := len(request.Reason); length != 0 && (length < 3 || length > 512) {
		return request, "", errors.New("approval comment must contain 3..512 characters")
	}
	if principal.Username == doc.CreatedBy {
		return request, "admin_self", nil
	}
	return request, "independent", nil
}

func recordEmergencyApproval(payload map[string]any, principal *middleware.SecurityContext, request approvalRequest, mode string) {
	payload["approval_mode"] = mode
	payload["approved_by"] = principal.Username
	payload["approved_at"] = time.Now().UTC()
	delete(payload, "approval_reason")
	if reason := strings.TrimSpace(request.Reason); reason != "" {
		payload["approval_reason"] = reason
	}
	if mode != "emergency_admin_self" {
		return
	}
	payload["emergency_self_approval"] = true
	payload["emergency_approved_by"] = principal.Username
	payload["emergency_approved_at"] = time.Now().UTC()
	payload["emergency_approval_reason"] = strings.TrimSpace(request.Reason)
	payload["emergency_approved_revision"] = request.ExpectedRevision
}
