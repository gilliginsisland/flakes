package flagutil

import (
	"fmt"
	"net"
)

// HostPort stores a validated host:port string
type HostPort string

// UnmarshalText validates and sets the host:port value.
func (hp *HostPort) UnmarshalText(text []byte) error {
	address := string(text)
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}

	// Reconstruct to normalize format (ensures valid IPv6 brackets)
	*hp = HostPort(net.JoinHostPort(host, port))
	return nil
}

// UnmarshalFlag calls UnmarshalText for go-flags compatibility.
func (hp *HostPort) UnmarshalFlag(value string) error {
	return hp.UnmarshalText([]byte(value))
}
