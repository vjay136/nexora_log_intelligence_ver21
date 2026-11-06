//go:build !windows

package output

import (
	"os"
	"syscall"
)

// PreserveFileAccess prevents a privileged replay command's atomic replacement
// from making an agent-owned queue or delivery journal unwritable by the agent.
func PreserveFileAccess(source, target string) error {
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err = os.Chmod(target, info.Mode().Perm()); err != nil {
		return err
	}
	return PreserveFileOwner(source, target)
}
func PreserveFileOwner(source, target string) error {
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	original, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	current, err := os.Stat(target)
	if err != nil {
		return err
	}
	owner, ok := current.Sys().(*syscall.Stat_t)
	if ok && (owner.Uid != original.Uid || owner.Gid != original.Gid) {
		return os.Chown(target, int(original.Uid), int(original.Gid))
	}
	return nil
}
