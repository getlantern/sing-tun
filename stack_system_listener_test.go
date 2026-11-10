package tun

import (
	"net"
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
