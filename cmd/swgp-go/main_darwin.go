//go:build darwin

package main

import (
	"context"
	"fmt"
	"github.com/database64128/swgp-go/service"
	"github.com/database64128/swgp-go/tslog"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"
	"unsafe"
	
	"golang.org/x/sys/unix"
)

// InterfaceConfig holds the network interface information for source routing
type InterfaceConfig struct {
	mu      sync.RWMutex
	ifname  string
	ifindex uint32
	ipv4    netip.Addr
	ipv6    netip.Addr
	logger  *tslog.Logger
}

// NewInterfaceConfig creates a new configuration for the specified interface
func NewInterfaceConfig(ifname string, logger *tslog.Logger) (*InterfaceConfig, error) {
	ic := &InterfaceConfig{
		ifname: ifname,
		logger: logger,
	}
	
	if err := ic.updateInterfaceInfo(); err != nil {
		return nil, err
	}
	
	return ic, nil
}

// updateInterfaceInfo gets the current IP and interface index for the specified interface
func (ic *InterfaceConfig) updateInterfaceInfo() error {
	iface, err := net.InterfaceByName(ic.ifname)
	if err != nil {
		return fmt.Errorf("getting interface %s: %w", ic.ifname, err)
	}
	
	ic.mu.Lock()
	defer ic.mu.Unlock()
	
	ic.ifindex = uint32(iface.Index)
	
	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("getting addresses for interface %s: %w", ic.ifname, err)
	}
	
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		
		if ipnet.IP.To4() != nil {
			// IPv4 address
			ic.ipv4, _ = netip.AddrFromSlice(ipnet.IP)
			ic.logger.Info("Found IPv4 address for interface", 
				slog.String("interface", ic.ifname),
				slog.String("ipv4", ic.ipv4.String()))
		} else {
			// IPv6 address - skip link-local addresses
			if !ipnet.IP.IsLinkLocalUnicast() {
				ic.ipv6, _ = netip.AddrFromSlice(ipnet.IP)
				ic.logger.Info("Found IPv6 address for interface", 
					slog.String("interface", ic.ifname),
					slog.String("ipv6", ic.ipv6.String()))
			}
		}
	}
	
	if ic.ipv4 == (netip.Addr{}) && ic.ipv6 == (netip.Addr{}) {
		return fmt.Errorf("no valid IP addresses found for interface %s", ic.ifname)
	}
	
	return nil
}

// GetSourceControlMessage returns a control message for the en0 interface
func (ic *InterfaceConfig) GetSourceControlMessage(destIP net.IP) []byte {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	
	var cmsg []byte
	
	if destIP.To4() != nil && ic.ipv4 != (netip.Addr{}) {
		// IPv4
		cmsg = make([]byte, unix.CmsgSpace(unix.SizeofInet4Pktinfo))
		cmsgHdr := (*unix.Cmsghdr)(unsafe.Pointer(&cmsg[0]))
		cmsgHdr.Level = unix.IPPROTO_IP
		cmsgHdr.Type = unix.IP_PKTINFO
		cmsgHdr.Len = unix.CmsgLen(unix.SizeofInet4Pktinfo)
		
		pktInfo := (*unix.Inet4Pktinfo)(unsafe.Pointer(&cmsg[unix.CmsgLen(0)]))
		copy(pktInfo.Spec_dst[:], ic.ipv4.AsSlice())
		pktInfo.Ifindex = ic.ifindex
	} else if ic.ipv6 != (netip.Addr{}) {
		// IPv6
		cmsg = make([]byte, unix.CmsgSpace(unix.SizeofInet6Pktinfo))
		cmsgHdr := (*unix.Cmsghdr)(unsafe.Pointer(&cmsg[0]))
		cmsgHdr.Level = unix.IPPROTO_IPV6
		cmsgHdr.Type = unix.IPV6_PKTINFO
		cmsgHdr.Len = unix.CmsgLen(unix.SizeofInet6Pktinfo)
		
		pktInfo := (*unix.Inet6Pktinfo)(unsafe.Pointer(&cmsg[unix.CmsgLen(0)]))
		copy(pktInfo.Addr[:], ic.ipv6.AsSlice())
		pktInfo.Ifindex = ic.ifindex
	}
	
	return cmsg
}

// Start periodically updates the interface information to keep it current
func (ic *InterfaceConfig) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := ic.updateInterfaceInfo(); err != nil {
					ic.logger.Warn("Failed to update interface info", tslog.Err(err))
				}
			}
		}
	}()
}

var ifConfig *InterfaceConfig
var initOnce sync.Once

// GetSourceRoutingControlMessage returns the control message for routing
func GetSourceRoutingControlMessage(destIP net.IP) []byte {
	if ifConfig == nil {
		return nil
	}
	return ifConfig.GetSourceControlMessage(destIP)
}

// initHook performs necessary initialization
func initHook(cfg *service.Config, logger *tslog.Logger) {
	initOnce.Do(func() {
		var err error
		
		// Create the interface configuration for en0
		ifConfig, err = NewInterfaceConfig("en0", logger)
		if err != nil {
			logger.Error("Failed to create interface config", tslog.Err(err))
			return
		}
		
		// Perform an initial update to ensure we have valid interface data
		if err := ifConfig.updateInterfaceInfo(); err != nil {
			logger.Warn("Initial interface update failed, will retry in background", tslog.Err(err))
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		service.RegisterCleanupHandler(func() {
			cancel()
		})
		
		// Start the periodic updates
		ifConfig.Start(ctx)
		
		// Create a wrapper function that uses our ifConfig to generate the control message
		sourceRoutingFunc := func(destIP []byte) []byte {
			return ifConfig.GetSourceControlMessage(destIP)
		}
		
		// Register the control message function with the service package
		service.SetSourceRoutingControlMessageFunc(sourceRoutingFunc)
		
		logger.Info("Source routing using interface configured", 
			slog.String("interface", "en0"),
			slog.Uint32("ifindex", ifConfig.ifindex),
			slog.String("ipv4", ifConfig.ipv4.String()),
			slog.String("ipv6", ifConfig.ipv6.String()))
	})
}

// cleanupHook performs necessary cleanup
func cleanupHook() {
	// Nothing to do here, the cleanup is handled by the context cancellation
}
