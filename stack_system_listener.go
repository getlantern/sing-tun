package tun

import (
	"context"
	"errors"
	"net"
	"reflect"
	"sync"
	"syscall"
	"time"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
)

const (
	relistenAttempts = 10
	relistenDelay    = 100 * time.Millisecond
)

// forwarderListener is the system stack's TCP forwarder listener. Android's
// netd destroys every socket bound to an address when that address is removed,
// and each VPN session's tun briefly removes and re-adds its address while it
// comes up. When netd handles that removal after the forwarder has bound, it
// destroys the listener, so Accept listens again on the same address and port.
// Keeping the port keeps the NAT rewrite on the packet path unchanged.
type forwarderListener struct {
	ctx     context.Context
	logger  logger.Logger
	network string
	address string
	config  net.ListenConfig

	access   sync.Mutex
	listener net.Listener
	closed   bool
}

func newForwarderListener(ctx context.Context, logger logger.Logger, config net.ListenConfig, network string, listener net.Listener) *forwarderListener {
	return &forwarderListener{
		ctx:      ctx,
		logger:   logger,
		network:  network,
		address:  listener.Addr().String(),
		config:   config,
		listener: listener,
	}
}

// Accept accepts on the current listener, listening again when the listener
// was destroyed under it.
func (f *forwarderListener) Accept() (net.Conn, error) {
	for {
		listener := f.current()
		if listener == nil {
			return nil, net.ErrClosed
		}
		conn, err := listener.Accept()
		if err == nil {
			return conn, nil
		}
		if !listenerDestroyed(err) {
			return nil, err
		}
		f.logger.Warn(f.network, " forwarder listener was destroyed (", errorWithoutAddress(err), "), listening again")
		relistenErr := f.relisten(listener)
		if relistenErr != nil {
			return nil, E.Errors(err, relistenErr)
		}
	}
}

func (f *forwarderListener) current() net.Listener {
	f.access.Lock()
	defer f.access.Unlock()
	if f.closed {
		return nil
	}
	return f.listener
}

// relisten replaces dead with a new listener on the same address. The address
// may be missing briefly while the tun's address flaps, so it retries.
func (f *forwarderListener) relisten(dead net.Listener) error {
	_ = dead.Close()
	var err error
	for range relistenAttempts {
		f.access.Lock()
		if f.closed {
			f.access.Unlock()
			return net.ErrClosed
		}
		var listener net.Listener
		listener, err = f.config.Listen(f.ctx, f.network, f.address)
		if err == nil {
			f.listener = listener
			f.access.Unlock()
			return nil
		}
		f.access.Unlock()
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		case <-time.After(relistenDelay):
		}
	}
	return E.New(f.network, " forwarder could not listen again: ", errorWithoutAddress(err))
}

func (f *forwarderListener) Close() error {
	if f == nil {
		return nil
	}
	f.access.Lock()
	defer f.access.Unlock()
	f.closed = true
	return f.listener.Close()
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
