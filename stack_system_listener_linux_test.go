//go:build linux

package tun

import (
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/sagernet/sing/common/logger"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// Shutting down a listening socket leaves it out of the LISTEN state, so its
// Accept fails with EINVAL as it does after netd destroys it.
func TestForwarderListenerListensAgainAfterDestroy(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	forwarder := newForwarderListener(t.Context(), logger.NOP(), "", net.ListenConfig{}, "tcp4", listener)
	defer forwarder.Close()

	accepted := make(chan net.Conn, 1)
	failed := make(chan error, 1)
	go func() {
		conn, err := forwarder.Accept()
		if err != nil {
			failed <- err
			return
		}
		accepted <- conn
	}()

	rawConn, err := listener.(syscall.Conn).SyscallConn()
	require.NoError(t, err)
	require.NoError(t, rawConn.Control(func(fd uintptr) {
		require.NoError(t, unix.Shutdown(int(fd), unix.SHUT_RD))
	}))
	require.Eventually(t, func() bool { return forwarder.current() != listener }, time.Second, 10*time.Millisecond)
	require.Equal(t, address, forwarder.current().Addr().String())

	client, err := net.DialTimeout("tcp4", address, time.Second)
	require.NoError(t, err)
	defer client.Close()
	select {
	case conn := <-accepted:
		conn.Close()
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("the listener was not replaced")
	}
}
