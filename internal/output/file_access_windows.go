//go:build windows

package output

import "os"

// New files inherit the destination directory ACL on Windows.
func PreserveFileAccess(source, target string) error {
	info, err := os.Stat(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.Chmod(target, info.Mode().Perm())
}
func PreserveFileOwner(string, string) error { return nil }
