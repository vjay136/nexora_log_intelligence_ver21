//go:build !windows

package output

import (
	"golang.org/x/sys/unix"
	"os"
)

func lockDeliveryHandle(file *os.File, nonblocking bool) error {
	flags := unix.LOCK_EX
	if nonblocking {
		flags |= unix.LOCK_NB
	}
	return unix.Flock(int(file.Fd()), flags)
}
