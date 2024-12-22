package gateway

import "net/netip"

// DetectOutgoingIface returns the outgoing interface information for the default route
// Returns the interface's IP address, interface index, and whether the detection was successful
func DetectOutgoingIface() (addr netip.Addr, ifindex uint32, ok bool) {
	return detectOutgoingIface()
}
