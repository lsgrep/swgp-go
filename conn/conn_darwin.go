package conn

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func setOutboundInterface(fd int, network string, _ *SocketInfo) error {
	// We're using control messages instead of IP_BOUND_IF
	return nil
}

func setRecvPktinfo(fd int, network string, _ *SocketInfo) error {
	switch network {
	case "udp4":
		if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_RECVPKTINFO, 1); err != nil {
			return fmt.Errorf("failed to set socket option IP_PKTINFO: %w", err)
		}
	case "udp6":
		if err := unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_RECVPKTINFO, 1); err != nil {
			return fmt.Errorf("failed to set socket option IPV6_RECVPKTINFO: %w", err)
		}
	default:
		return fmt.Errorf("unsupported network: %s", network)
	}
	return nil
}

func (fns setFuncSlice) appendSetOutboundInterface() setFuncSlice {
	// We're using control messages instead of IP_BOUND_IF
	return fns
}

func setDontFrag(fd int, network string, _ *SocketInfo) error {
	switch network {
	case "udp4":
		if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_DONTFRAG, 1); err != nil {
			return fmt.Errorf("failed to set socket option IP_DONTFRAG: %w", err)
		}
	case "udp6":
		if err := unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_DONTFRAG, 1); err != nil {
			return fmt.Errorf("failed to set socket option IPV6_DONTFRAG: %w", err)
		}
	default:
		return fmt.Errorf("unsupported network: %s", network)
	}
	return nil
}

func (fns setFuncSlice) appendSetDontFrag() setFuncSlice {
	return append(fns, setDontFrag)
}

func (lso ListenerSocketOptions) buildSetFns() setFuncSlice {
	return setFuncSlice{}.
		appendGetIPv6Only().
		appendSetSendBufferSize(lso.SendBufferSize).
		appendSetRecvBufferSize(lso.ReceiveBufferSize).
		appendSetTrafficClassFunc(lso.TrafficClass).
		appendSetPMTUDFunc(lso.PathMTUDiscovery).
		appendSetRecvPktinfoFunc(lso.ReceivePacketInfo).
		appendSetDontFrag().
		appendSetOutboundInterface()
}
