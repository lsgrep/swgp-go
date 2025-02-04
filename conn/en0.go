package conn

import (
	"net"
	"net/netip"
)

// GetHardcodedEn0Info returns the hardcoded interface index and IP address for en0.
// This is used to ensure stable communication on macOS.
func GetHardcodedEn0Info() (uint32, netip.Addr, error) {
	intf, err := net.InterfaceByName("en0")
	if err != nil {
		return 0, netip.Addr{}, err
	}

	addrs, err := intf.Addrs()
	if err != nil {
		return 0, netip.Addr{}, err
	}

	// Find IPv4 address
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return uint32(intf.Index), netip.AddrFrom4([4]byte(ip4)), nil
			}
		}
	}

	return 0, netip.Addr{}, nil
}
