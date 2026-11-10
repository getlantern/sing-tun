package tun

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sagernet/sing-tun/internal/gtcpip/header"
	"github.com/sagernet/sing/common/logger"

	"github.com/stretchr/testify/require"
)

func TestErrorWithoutAddressDropsAddresses(t *testing.T) {
	err := &net.OpError{
		Op:     "accept",
		Net:    "tcp",
		Source: &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 443},
		Addr:   &net.TCPAddr{IP: net.IPv4(198, 51, 100, 9), Port: 10001},
		Err:    os.NewSyscallError("accept4", syscall.EMFILE),
	}
	require.Contains(t, err.Error(), "198.51.100.9")
	description := errorWithoutAddress(fmt.Errorf("wrapped: %w", err))
	require.Equal(t, syscall.EMFILE.Error(), description)

	closed := &net.OpError{Op: "accept", Net: "tcp", Addr: err.Addr, Err: net.ErrClosed}
	require.NotContains(t, errorWithoutAddress(closed), "198.51.100.9")

	other := &net.OpError{Op: "raw-control", Net: "tcp", Addr: err.Addr, Err: os.ErrInvalid}
	require.NotContains(t, errorWithoutAddress(other), "198.51.100.9")
}

func TestLogUnacceptedResetIsRateLimited(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	log := &debugCounter{}
	forwarder := newForwarderListener(t.Context(), log, net.ListenConfig{}, "tcp4", listener)
	stack := &System{logger: log, tunName: "tun-test0"}
	for range 100 {
		stack.logUnacceptedReset("tcp4", forwarder, 10001)
	}
	require.Eventually(t, func() bool { return log.count() == 1 }, time.Second, 10*time.Millisecond)
	require.Never(t, func() bool { return log.count() > 1 }, 200*time.Millisecond, 10*time.Millisecond)
	require.NotContains(t, log.last(), "127.0.0.1")

	stack.lastUnacceptedReset.Store(0)
	stack.logUnacceptedReset("tcp4", forwarder, 10002)
	require.Eventually(t, func() bool { return log.count() == 2 }, time.Second, 10*time.Millisecond)
}

type debugCounter struct {
	logger.Logger
	access   sync.Mutex
	messages []string
}

func (l *debugCounter) Debug(args ...any) {
	l.access.Lock()
	defer l.access.Unlock()
	l.messages = append(l.messages, fmt.Sprint(args...))
}

func (l *debugCounter) count() int {
	l.access.Lock()
	defer l.access.Unlock()
	return len(l.messages)
}

func (l *debugCounter) last() string {
	l.access.Lock()
	defer l.access.Unlock()
	return strings.Join(l.messages[len(l.messages)-1:], "")
}

func TestOpeningSYNClearsAccepted(t *testing.T) {
	nat := &TCPNat{portMap: map[uint16]*TCPSession{10001: {}}}
	session := nat.portMap[10001]
	session.accepted.Store(true)

	segment := header.TCP(make([]byte, header.TCPMinimumSize))
	segment.Encode(&header.TCPFields{Flags: header.TCPFlagSyn | header.TCPFlagAck, DataOffset: header.TCPMinimumSize})
	require.False(t, isOpeningSYN(segment))
	segment.Encode(&header.TCPFields{Flags: header.TCPFlagSyn, DataOffset: header.TCPMinimumSize})
	require.True(t, isOpeningSYN(segment))

	nat.markOpening(10001)
	require.False(t, session.accepted.Load())
	nat.markOpening(10002)
}
