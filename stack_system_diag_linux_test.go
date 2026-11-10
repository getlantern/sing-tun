//go:build linux

package tun

import (
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestKernelInterfaceIndexMatchesNet(t *testing.T) {
	loopback, err := net.InterfaceByName("lo")
	require.NoError(t, err)
	index, err := kernelInterfaceIndex("lo")
	require.NoError(t, err)
	require.Equal(t, loopback.Index, index)

	_, err = kernelInterfaceIndex("no-such-if0")
	require.Error(t, err)
}

func TestSocketBoundInterface(t *testing.T) {
	unbound, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer unbound.Close()
	index, err := socketBoundInterface(unbound)
	require.NoError(t, err)
	require.Zero(t, index)

	loopback, err := net.InterfaceByName("lo")
	require.NoError(t, err)
	var config net.ListenConfig
	config.Control = func(_, _ string, conn syscall.RawConn) error {
		var sockErr error
		err := conn.Control(func(fd uintptr) {
			sockErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_BINDTOIFINDEX, loopback.Index)
		})
		if err != nil {
			return err
		}
		return sockErr
	}
	bound, err := config.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer bound.Close()
	index, err = socketBoundInterface(bound)
	require.NoError(t, err)
	require.Equal(t, loopback.Index, index)
}
