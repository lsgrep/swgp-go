//go:build darwin

package gateway

import (
	"net/netip"
	"syscall"

	"golang.org/x/net/route"
)

func detectOutgoingIface() (addr netip.Addr, ifindex uint32, ok bool) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	for _, m := range msgs {
		rm, ok := m.(*route.RouteMessage)
		if !ok {
			continue
		}

		// Look for default route (destination 0.0.0.0/0)
		if len(rm.Addrs) <= syscall.RTAX_DST || len(rm.Addrs) <= syscall.RTAX_GATEWAY {
			continue
		}

		dst := rm.Addrs[syscall.RTAX_DST]
		gw := rm.Addrs[syscall.RTAX_GATEWAY]

		// Check if this is a default route (0.0.0.0/0)
		if dst == nil {
			continue
		}

		switch d := dst.(type) {
		case *route.Inet4Addr:
			// Check if this is the default route (0.0.0.0)
			if d.IP != [4]byte{0, 0, 0, 0} {
				continue
			}

			// Found default route, get gateway
			if gw == nil {
				continue
			}

			switch g := gw.(type) {
			case *route.Inet4Addr:
				ip := netip.AddrFrom4([4]byte(g.IP))
				if !ip.IsLoopback() {
					return ip, uint32(rm.Index), true
				}
			}
		case *route.Inet6Addr:
			// Check if this is the default route (::/0)
			if !isZeroIPv6(d.IP) {
				continue
			}

			// Found default route, get gateway
			if gw == nil {
				continue
			}

			switch g := gw.(type) {
			case *route.Inet6Addr:
				var ipBytes [16]byte
				copy(ipBytes[:], g.IP[:])
				ip := netip.AddrFrom16(ipBytes)
				if !ip.IsLoopback() {
					return ip, uint32(rm.Index), true
				}
			}
		}
	}

	return netip.Addr{}, 0, false
}

func isZeroIPv6(ip [16]byte) bool {
	for _, b := range ip {
		if b != 0 {
			return false
		}
	}
	return true
}
