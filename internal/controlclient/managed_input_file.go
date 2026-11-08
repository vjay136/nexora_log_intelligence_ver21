package controlclient

import (
	"encoding/json"
	"fmt"

	"nexora.local/nli-v20/internal/protocol"
)

func (c *Client) applyManagedInputFile(payload json.RawMessage) (string, error) {
	var request protocol.ManagedInputFileRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return "", fmt.Errorf("invalid managed input file request: %w", err)
	}
	return manageInputFile(protocol.ManagedInputDirectory, c.cfg.AgentConfigPath, request)
}
