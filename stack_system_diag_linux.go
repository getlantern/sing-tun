//go:build linux

package tun

import (
	"net"
	"syscall"

	E "github.com/sagernet/sing/common/exceptions"

	"golang.org/x/sys/unix"
)

// socketBoundInterface returns the interface index the listener's socket is
// bound to, or 0 when it isn't bound to one.
func socketBoundInterface(listener net.Listener) (int, error) {
	syscallListener, ok := listener.(syscall.Conn)
	if !ok {
		return 0, E.New("listener has no file descriptor")
	}
	rawConn, err := syscallListener.SyscallConn()
	if err != nil {
		return 0, err
	}
	var index int
	var sockErr error
	err = rawConn.Control(func(fd uintptr) {
		index, sockErr = unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_BINDTOIFINDEX)
	})
	if err != nil {
		return 0, err
	}
	return index, sockErr
}

// kernelInterfaceIndex asks the kernel for the named interface's index with an
// ioctl rather than netlink, which Android denies to apps.
func kernelInterfaceIndex(name string) (int, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return 0, err
	}
	defer unix.Close(fd)
	ifreq, err := unix.NewIfreq(name)
	if err != nil {
		return 0, err
	}
	err = unix.IoctlIfreq(fd, unix.SIOCGIFINDEX, ifreq)
	if err != nil {
		return 0, err
	}
	return int(ifreq.Uint32()), nil
}
