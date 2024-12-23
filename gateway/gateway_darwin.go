//go:build darwin

package gateway

import (
	"net/netip"
	"syscall"

	"golang.org/x/net/route"
)

func detectOutgoingIface() (addr netip.Addr, ifindex uint32, ok bool) {
	// Try IPv4 first
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_IFLIST, 0)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_IFLIST, rib)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	for _, m := range msgs {
		switch msg := m.(type) {
		case *route.InterfaceMessage:
			if msg.Flags&syscall.IFF_UP == 0 || msg.Flags&syscall.IFF_LOOPBACK != 0 {
				continue
			}
			for _, a := range msg.Addrs {
				switch addr := a.(type) {
				case *route.Inet4Addr:
					ip := netip.AddrFrom4(addr.IP)
					if !ip.IsLoopback() && !ip.IsUnspecified() {
						return ip, uint32(msg.Index), true
					}
				}
			}
		}
	}

	// Try IPv6 if IPv4 failed
	rib, err = route.FetchRIB(syscall.AF_INET6, syscall.NET_RT_IFLIST, 0)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	msgs, err = route.ParseRIB(syscall.NET_RT_IFLIST, rib)
	if err != nil {
		return netip.Addr{}, 0, false
	}

	for _, m := range msgs {
		switch msg := m.(type) {
		case *route.InterfaceMessage:
			if msg.Flags&syscall.IFF_UP == 0 || msg.Flags&syscall.IFF_LOOPBACK != 0 {
				continue
			}
			for _, a := range msg.Addrs {
				switch addr := a.(type) {
				case *route.Inet6Addr:
					var ipBytes [16]byte
					copy(ipBytes[:], addr.IP[:])
					ip := netip.AddrFrom16(ipBytes)
					if !ip.IsLoopback() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() {
						return ip, uint32(msg.Index), true
					}
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
