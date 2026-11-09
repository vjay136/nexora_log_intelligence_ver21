package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnsureHourlyFiles creates missing local hourly destinations without opening
// their writers or delivering a record. The caller supplies one time for the
// entire set so the files all belong to the same UTC hour.
func EnsureHourlyFiles(configs []Config, now time.Time) error {
	var failures []error
	for _, cfg := range configs {
		if cfg.Type != "file" || cfg.PathTemplate == "" {
			continue
		}
		path, err := resolveFilePath(cfg, now)
		if err == nil {
			err = ensureEmptyHourlyFile(path)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("prepare hourly output %q: %w", cfg.PathTemplate, err))
		}
	}
	return errors.Join(failures...)
}

func ensureEmptyHourlyFile(path string) error {
	dir := filepath.Dir(path)
	if err := makeOwnedDirectory(dir, 0750); err != nil {
		return err
	}
	// Exclusive creation leaves live output, replay output, and existing files
	// intact even if a writer reaches this hour at the same time.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if errors.Is(err, os.ErrExist) {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output path is not a regular file")
		}
		return nil
	}
	if err != nil {
		return err
	}
	ownerErr := PreserveFileOwner(dir, path)
	closeErr := file.Close()
	return errors.Join(ownerErr, closeErr, syncDeliveryParents(dir))
}
