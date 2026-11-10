package tun

import (
	"net"
	"strconv"
)

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
		return "unknown (" + err.Error() + ")"
	}
	return strconv.Itoa(index)
}
