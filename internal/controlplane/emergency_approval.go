package controlplane

import (
	"fmt"
	"strings"
	"time"

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

// approvalMode preserves independent approval as the normal path. A creator
// may approve their own parser change only via the administrator-only recovery
// path with a reason and exact revision/digest binding. Ver21 deliberately
// makes this capability intrinsic to the admin role: an operator must not be
// stranded merely because a deployment omitted an optional feature flag.
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
	if !principalHasPermission(principal, middleware.PermissionEmergencyApprove) {
		return "", nil, fmt.Errorf("creator cannot approve own change; emergency administrator permission is required")
	}
	if !request.EmergencySelfApproval {
		return "", nil, fmt.Errorf("creator cannot approve own change; explicitly request emergency_self_approval with reason and exact revision/hash")
	}
	reason := strings.TrimSpace(request.Reason)
	if len(reason) < 12 || len(reason) > 512 {
		return "", nil, fmt.Errorf("emergency self-approval reason must contain 12..512 characters")
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

func recordEmergencyApproval(payload map[string]any, principal *middleware.SecurityContext, request approvalRequest, mode string) {
	if mode != "emergency_admin_self" {
		payload["approval_mode"] = mode
		if reason := strings.TrimSpace(request.Reason); reason != "" {
			payload["approval_reason"] = reason
		}
		return
	}
	payload["approval_mode"] = mode
	payload["emergency_self_approval"] = true
	payload["emergency_approved_by"] = principal.Username
	payload["emergency_approved_at"] = time.Now().UTC()
	payload["emergency_approval_reason"] = strings.TrimSpace(request.Reason)
	payload["emergency_approved_revision"] = request.ExpectedRevision
}
