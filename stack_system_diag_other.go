//go:build !linux

package tun

import (
	"errors"
	"net"
)

func socketBoundInterface(net.Listener) (int, error) {
	return 0, errors.ErrUnsupported
}

func kernelInterfaceIndex(string) (int, error) {
	return 0, errors.ErrUnsupported
}
