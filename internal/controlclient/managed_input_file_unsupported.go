//go:build !linux

package controlclient

import (
	"fmt"

	"nexora.local/nli-v20/internal/protocol"
)

func listManagedInputFiles(_ string) ([]string, bool) { return nil, false }

func manageInputFile(_, _ string, _ protocol.ManagedInputFileRequest) (string, error) {
	return "", fmt.Errorf("managed input file operations require Linux")
}
