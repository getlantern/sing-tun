package tun

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/sagernet/sing/common/logger"

	"github.com/stretchr/testify/require"
)

func TestForwarderListenerCloseEndsAccept(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	forwarder := newForwarderListener(t.Context(), logger.NOP(), net.ListenConfig{}, "tcp4", listener)
	accepted := make(chan error, 1)
	go func() {
		_, err := forwarder.Accept()
		accepted <- err
	}()
	require.NoError(t, forwarder.Close())
	require.ErrorIs(t, <-accepted, net.ErrClosed)
	_, err = forwarder.Accept()
	require.ErrorIs(t, err, net.ErrClosed)
	require.ErrorIs(t, forwarder.relisten(listener), net.ErrClosed)

	var nilForwarder *forwarderListener
	require.NoError(t, nilForwarder.Close())
}

func TestErrorWithoutAddressDropsAddresses(t *testing.T) {
	err := &net.OpError{
		Op:     "accept",
		Net:    "tcp",
		Source: &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 443},
		Addr:   &net.TCPAddr{IP: net.IPv4(198, 51, 100, 9), Port: 10001},
		Err:    os.NewSyscallError("accept4", syscall.EINVAL),
	}
	require.Contains(t, err.Error(), "198.51.100.9")
	require.Equal(t, syscall.EINVAL.Error(), errorWithoutAddress(fmt.Errorf("wrapped: %w", err)))

	closed := &net.OpError{Op: "accept", Net: "tcp", Addr: err.Addr, Err: net.ErrClosed}
	require.NotContains(t, errorWithoutAddress(closed), "198.51.100.9")

	other := &net.OpError{Op: "listen", Net: "tcp", Addr: err.Addr, Err: os.ErrInvalid}
	require.NotContains(t, errorWithoutAddress(other), "198.51.100.9")
}
