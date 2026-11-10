//go:build !windows

package tun

import (
	"errors"

	"golang.org/x/sys/unix"
)

func fixWindowsFirewall() error {
	return nil
}

func retryableListenError(err error) bool {
	return errors.Is(err, unix.EADDRNOTAVAIL)
}

// listenerDestroyed reports whether Accept failed because the listening socket
// stopped listening, as when Android's netd destroys it.
func listenerDestroyed(err error) bool {
	return errors.Is(err, unix.EINVAL)
}
