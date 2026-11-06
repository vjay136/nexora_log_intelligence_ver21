//go:build windows

package output

import (
	"golang.org/x/sys/windows"
	"os"
)

func lockDeliveryHandle(file *os.File, nonblocking bool) error {
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK)
	if nonblocking {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	return windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, new(windows.Overlapped))
}
