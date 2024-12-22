//go:build !darwin

package gateway

import "net/netip"

func detectOutgoingIface() (addr netip.Addr, ifindex uint32, ok bool) {
	return netip.Addr{}, 0, false
}
