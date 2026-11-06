//go:build !windows

package agent

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
)

func lockAgentState(dir string) (func() error, error) {
	f, err := os.OpenFile(filepath.Join(dir, ".agent.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, err
	}
	return f.Close, nil
}
