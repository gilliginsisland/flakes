package flagutil

import (
	"fmt"
	"net"
)

// HostPort stores a validated host:port string
type HostPort string

// Set validates and sets the host:port value
func (hp *HostPort) Set(value string) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}

	// Reconstruct to normalize format (ensures valid IPv6 brackets)
	*hp = HostPort(net.JoinHostPort(host, port))
	return nil
}

// String returns the stored host:port value
func (hp *HostPort) String() string {
	if hp != nil {
		return string(*hp)
	}
	return ""
}
