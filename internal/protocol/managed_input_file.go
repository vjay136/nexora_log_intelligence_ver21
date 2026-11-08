package protocol

import (
	"fmt"
	"regexp"
)

const (
	ManagedInputDirectory   = "/var/log/nli/input"
	ManagedInputFileCommand = "managed_input_file"
)

type ManagedInputFileRequest struct {
	Action      string `json:"action"`
	Source      string `json:"source"`
	Destination string `json:"destination,omitempty"`
}

var managedInputName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,127}$`)

func (r ManagedInputFileRequest) Validate() error {
	if !managedInputName.MatchString(r.Source) {
		return fmt.Errorf("source must be a direct managed input filename")
	}
	switch r.Action {
	case "copy":
		if !managedInputName.MatchString(r.Destination) || r.Source == r.Destination {
			return fmt.Errorf("copy requires a different direct managed input destination filename")
		}
	case "delete":
		if r.Destination != "" {
			return fmt.Errorf("delete does not accept a destination")
		}
	default:
		return fmt.Errorf("unsupported managed input file action")
	}
	return nil
}
