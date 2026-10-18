//go:build with_gvisor

package tun

import (
	"testing"

	"github.com/sagernet/gvisor/pkg/tcpip/stack"
)

type recordingLinkEndpoint struct {
	stack.LinkEndpoint
	dispatcher stack.NetworkDispatcher
}

func (e *recordingLinkEndpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher
}

func TestLinkEndpointFilterDetach(t *testing.T) {
	endpoint := &recordingLinkEndpoint{}
	filter := &LinkEndpointFilter{LinkEndpoint: endpoint}
	dispatcher := &struct{ stack.NetworkDispatcher }{}
	filter.Attach(dispatcher)
	wrapped, ok := endpoint.dispatcher.(*networkDispatcherFilter)
	if !ok || wrapped.NetworkDispatcher != dispatcher {
		t.Fatal("non-nil dispatcher was not forwarded through the packet filter")
	}

	// gVisor uses a literal nil to stop and join its packet reader. Wrapping
	// nil keeps the reader polling the TUN fd even after the tunnel closes.
	filter.Attach(nil)
	if endpoint.dispatcher != nil {
		t.Fatalf("detach passed %T to the underlying endpoint; want nil", endpoint.dispatcher)
	}
}
