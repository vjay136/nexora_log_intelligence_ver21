package agent

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func deliverySourceIdentity(nodeID, path string, state cursor) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d", nodeID, path, state.Device, state.Inode, state.Generation)))
	return fmt.Sprintf("%x", hash[:24])
}

// One physical source must have one cursor owner. Repeated paths, symlinks and
// hard links otherwise let independent workers count the same file twice.
func validateDistinctInputs(inputs []InputConfig) error {
	paths := map[string]bool{}
	var files []os.FileInfo
	for _, input := range inputs {
		path, err := filepath.Abs(filepath.Clean(input.Path))
		if err != nil {
			return err
		}
		key := path
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if paths[key] {
			return fmt.Errorf("duplicate input source: %s", input.Path)
		}
		paths[key] = true
		if info, err := os.Stat(path); err == nil {
			for _, prior := range files {
				if os.SameFile(prior, info) {
					return fmt.Errorf("input aliases refer to the same physical source: %s", input.Path)
				}
			}
			files = append(files, info)
		}
	}
	return nil
}
