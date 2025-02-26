package service

import (
	"net"
	"sync"
)

var (
	sourceRoutingMu      sync.RWMutex
	sourceRoutingEnabled bool
	sourceRoutingFunc    func(net.IP) []byte
)

// SetSourceRoutingControlMessageFunc sets the function that provides control messages for IP routing
func SetSourceRoutingControlMessageFunc(fn func(net.IP) []byte) {
	sourceRoutingMu.Lock()
	defer sourceRoutingMu.Unlock()
	
	if fn != nil {
		sourceRoutingFunc = fn
		sourceRoutingEnabled = true
	}
}

// GetSourceRoutingControlMessage returns a control message for the specified destination IP
// that will ensure the packet gets routed correctly when WireGuard is active
func GetSourceRoutingControlMessage(destIP net.IP) []byte {
	sourceRoutingMu.RLock()
	defer sourceRoutingMu.RUnlock()
	
	if !sourceRoutingEnabled || sourceRoutingFunc == nil {
		return nil
	}
	
	return sourceRoutingFunc(destIP)
}

// RegisterCleanupHandler registers a function to be called during cleanup
func RegisterCleanupHandler(fn func()) {
	// Store the cleanup handler to be called later
	if fn != nil {
		cleanupHandlers = append(cleanupHandlers, fn)
	}
}

// List of cleanup handlers
var cleanupHandlers []func()

// RunCleanupHandlers calls all registered cleanup handlers
func RunCleanupHandlers() {
	for _, handler := range cleanupHandlers {
		handler()
	}
	cleanupHandlers = nil
}
