package tun

import (
	"errors"
	"net"
	"reflect"
	"strconv"
	"syscall"
	"time"

	"github.com/sagernet/sing-tun/internal/gtcpip/header"
)

// unacceptedResetLogInterval bounds how often a reset the forwarder never
// accepted is logged, since a broken forwarder resets every new connection.
const unacceptedResetLogInterval = 5 * time.Second

// logUnacceptedReset logs that the kernel reset a connection the forwarder
// never accepted. It runs on the packet path, so it does at most one check
// there and gathers the interface indexes on its own goroutine.
func (s *System) logUnacceptedReset(network string, listener *forwarderListener, natPort uint16) {
	now := time.Now().UnixNano()
	last := s.lastUnacceptedReset.Load()
	if now-last < int64(unacceptedResetLogInterval) || !s.lastUnacceptedReset.CompareAndSwap(last, now) {
		return
	}
	go func() {
		current := listener.current()
		if current == nil {
			return
		}
		s.logger.Debug(network, " forwarder reset a connection it never accepted, nat port ", natPort, ": ", s.forwarderInterfaces(current))
	}()
}

// isOpeningSYN reports whether the segment opens a connection. A connection is
// accepted only once its handshake completes, so a retransmitted SYN never
// belongs to an accepted one.
func isOpeningSYN(tcpHdr header.TCP) bool {
	flags := tcpHdr.Flags()
	return flags.Contains(header.TCPFlagSyn) && !flags.Contains(header.TCPFlagAck)
}

// forwarderInterfaces compares the interface the TCP forwarder's listener is
// bound to with the tun's current kernel index, so a listener left bound to an
// earlier tun shows up in the log. It names no addresses.
func (s *System) forwarderInterfaces(listener net.Listener) string {
	bound, err := socketBoundInterface(listener)
	description := "bound interface " + describeInterfaceIndex(bound, err)
	tunIndex, err := kernelInterfaceIndex(s.tunName)
	description += ", " + s.tunName + " index " + describeInterfaceIndex(tunIndex, err)
	if s.interfaceFinder != nil {
		var finderIndex int
		iif, err := s.interfaceFinder.ByName(s.tunName)
		if err == nil {
			finderIndex = iif.Index
		}
		description += ", finder index " + describeInterfaceIndex(finderIndex, err)
	}
	return description
}

func describeInterfaceIndex(index int, err error) string {
	if err != nil {
		return "unknown (" + errorWithoutAddress(err) + ")"
	}
	return strconv.Itoa(index)
}

// errorWithoutAddress describes err by its errno, or failing that its type,
// because a net.OpError's text includes the socket's addresses.
func errorWithoutAddress(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno.Error()
	}
	if errors.Is(err, net.ErrClosed) {
		return "use of closed network connection"
	}
	return reflect.TypeOf(err).String()
}
