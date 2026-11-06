package output

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LockLocalFile coordinates short append/atomic-queue-merge transactions. It
// never locks customer file bytes, so file tail consumers remain independent.
// Callers must release it before network delivery or parser work.
func LockLocalFile(path string) (func() error, error) {
	return lockLocalOperation(path, "append", false)
}

// LockReplayOperation admits one replay command per queue without preventing
// the live agent from appending new evidence during remote delivery.
func LockReplayOperation(path string) (func() error, error) {
	return lockLocalOperation(path, "replay", true)
}

// DeliveryJournalPath is shared by live ingestion and explicit replay. Their
// identities differ, but append recovery must observe the same pending intent.
func DeliveryJournalPath(stateDir, path, template string) string {
	key := sha256.Sum256([]byte(path + "\x00" + template))
	return filepath.Join(stateDir, "delivery", fmt.Sprintf("%x.wal", key))
}

// LockLocalDelivery orders WAL ownership before the short output append/merge
// lock. Replay must release this pair before parsing or HTTP delivery.
func LockLocalDelivery(cfg Config, path string) (func() error, error) {
	if cfg.DeliveryStatePath == "" {
		return nil, fmt.Errorf("local delivery journal path is required")
	}
	journalUnlock, err := LockLocalFile(cfg.DeliveryStatePath)
	if err != nil {
		return nil, err
	}
	fileUnlock, err := LockLocalFile(path)
	if err != nil {
		_ = journalUnlock()
		return nil, err
	}
	return func() error { return errors.Join(fileUnlock(), journalUnlock()) }, nil
}

func lockLocalOperation(path, kind string, nonblocking bool) (func() error, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	directory := filepath.Dir(absolute)
	if err = makeOwnedDirectory(directory, 0750); err != nil {
		return nil, err
	}
	if resolved, resolveErr := filepath.EvalSymlinks(directory); resolveErr == nil {
		absolute = filepath.Join(resolved, filepath.Base(absolute))
		directory = resolved
	}
	key := sha256.Sum256([]byte(absolute))
	lockPath := filepath.Join(directory, fmt.Sprintf(".nli-%s-%x.lock", kind, key[:16]))
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	if err = PreserveFileOwner(directory, lockPath); err != nil {
		f.Close()
		return nil, err
	}
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return nil, err
	}
	if err = lockDeliveryHandle(f, nonblocking); err != nil {
		f.Close()
		return nil, fmt.Errorf("local %s operation is already active or cannot be locked: %w", kind, err)
	}
	return f.Close, nil
}

func makeOwnedDirectory(path string, mode os.FileMode) error {
	ancestor := path
	created := []string{}
	for {
		_, err := os.Stat(ancestor)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		created = append(created, ancestor)
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return err
		}
		ancestor = parent
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	for i := len(created) - 1; i >= 0; i-- {
		if err := PreserveFileOwner(ancestor, created[i]); err != nil {
			return err
		}
	}
	return nil
}
func lockLocalPaths(paths ...string) (func() error, error) {
	unique := map[string]bool{}
	ordered := []string{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if resolved, resolveErr := filepath.EvalSymlinks(filepath.Dir(absolute)); resolveErr == nil {
			absolute = filepath.Join(resolved, filepath.Base(absolute))
		}
		if !unique[absolute] {
			unique[absolute] = true
			ordered = append(ordered, absolute)
		}
	}
	sort.Strings(ordered)
	var unlockers []func() error
	unlock := func() error {
		var failures []error
		for i := len(unlockers) - 1; i >= 0; i-- {
			failures = append(failures, unlockers[i]())
		}
		return errors.Join(failures...)
	}
	for _, path := range ordered {
		release, err := LockLocalFile(path)
		if err != nil {
			_ = unlock()
			return nil, err
		}
		unlockers = append(unlockers, release)
	}
	return unlock, nil
}
