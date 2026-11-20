// Package bundlehistory retains bounded evidence of previously synchronized
// generations. History is never an authority for publication or activation.
package bundlehistory

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"nexora.local/nli-v20/internal/canonicaljson"
)

const Limit = 10
const maxBytes = 16 << 20

// Preserve must complete before replacing an active generation. The caller's
// verifier checks its canonical bundle digest and embedded asset hashes.
func Preserve(assetDir string, verify func([]byte) (string, error)) error {
	current := filepath.Join(assetDir, "bundle.json")
	data, err := read(current)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	hash, err := verify(data)
	if err != nil || !validHash(hash) {
		return fmt.Errorf("prior parser bundle cannot be archived: %v", err)
	}
	canonical, err := canonicaljson.Normalize(data)
	if err != nil {
		return err
	}
	dir := filepath.Join(filepath.Dir(assetDir), "parser-assets-history")
	if info, err := os.Lstat(dir); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return errors.New("parser history path is not a regular directory")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type entry struct {
		name     string
		modified time.Time
	}
	retained := []entry{}
	found := false
	for _, item := range entries {
		name := item.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		expected := strings.TrimSuffix(name, ".json")
		if !validHash(expected) {
			return errors.New("parser history contains an invalid archive name")
		}
		path := filepath.Join(dir, name)
		archived, err := read(path)
		if err != nil {
			return err
		}
		actual, err := verify(archived)
		if err != nil || actual != expected {
			return fmt.Errorf("parser history archive %s failed verification", name)
		}
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		retained = append(retained, entry{name, info.ModTime()})
		if actual == hash {
			found = true
		}
	}
	if !found {
		file, err := os.CreateTemp(dir, ".archive-*")
		if err != nil {
			return err
		}
		temp := file.Name()
		defer os.Remove(temp)
		if err = file.Chmod(0600); err == nil {
			_, err = file.Write(canonical)
		}
		if err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		name := hash + ".json"
		if err := os.Rename(temp, filepath.Join(dir, name)); err != nil {
			return err
		}
		if err := syncDirectory(dir); err != nil {
			return err
		}
		retained = append(retained, entry{name, time.Now()})
	}
	sort.Slice(retained, func(i, j int) bool {
		if retained[i].modified.Equal(retained[j].modified) {
			return retained[i].name < retained[j].name
		}
		return retained[i].modified.Before(retained[j].modified)
	})
	for len(retained) > Limit {
		index := 0
		// Preserve the generation about to be replaced even when it was archived
		// previously and its timestamp is older than other retained generations.
		if retained[index].name == hash+".json" {
			index++
		}
		if err := os.Remove(filepath.Join(dir, retained[index].name)); err != nil {
			return err
		}
		retained = append(retained[:index], retained[index+1:]...)
	}
	return syncDirectory(dir)
}

func syncDirectory(path string) error {
	// Directory fsync is unavailable through Windows os.File; production RPMs
	// run on Linux, where the archive rename must be durable before the swap.
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func validHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, errors.New("parser bundle archive is not a bounded regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if len(data) > maxBytes {
		return nil, errors.New("parser bundle archive exceeds limit")
	}
	return data, err
}
