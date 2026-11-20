package controlplane

import (
	"errors"
	"net/http"
	metastore "nexora.local/nli-v20/internal/store"
)

// Only static, credential-free explanations cross the browser boundary.
func metadataFailureDetails(err error) map[string]any {
	code, detail, action := "metadata_unavailable", "The authoritative metadata store could not be read. Existing records must not be treated as deleted.", "Check System Health and the controller's metadata connection, then refresh."
	var rpc *metastore.RPCError
	if errors.As(err, &rpc) {
		switch {
		case rpc.Code == "42501":
			code = "metadata_authentication_failed"
			detail = "Supabase rejected this controller's secret or RPC permissions. This is separate from your Webapp login."
			action = "Use the controller secret already registered for this Supabase project. Check /etc/nli/controller.env on the primary controller; do not reapply bootstrap from another node with a different secret."
		case rpc.Status == 401 || rpc.Status == 403:
			code, detail, action = "metadata_authentication_failed", "Supabase rejected the metadata API credentials.", "Verify the project URL and its publishable key on the controller, then check the registered controller secret."
		case rpc.Code == "PGRST202" || rpc.Code == "42883":
			code, detail, action = "metadata_schema_missing", "A required metadata RPC is missing or is absent from the API schema cache.", "Apply the matching generated bootstrap using this project's existing controller secret, then verify System Health."
		}
	}
	return map[string]any{"backend_error": code, "error_detail": detail, "action": action}
}

func replyMetadataFailure(w http.ResponseWriter, message string, err error) {
	w.Header().Set("Cache-Control", "no-store")
	value := metadataFailureDetails(err)
	value["error"] = message
	w.Header().Set("X-NLI-Backend-Error", value["backend_error"].(string))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	jsonOK(w, value)
}
