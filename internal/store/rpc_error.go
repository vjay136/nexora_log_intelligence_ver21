package store

import "fmt"

// RPCError preserves the database error code separately from the HTTP status.
// PostgREST returns 401 for controller-secret rejection under the anon role;
// that does not imply that the project's public API key is invalid.
type RPCError struct {
	Function string
	Status   int
	Code     string
	Message  string
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("Supabase %s failed with HTTP %d (%s): %s", e.Function, e.Status, e.Code, e.Message)
}
