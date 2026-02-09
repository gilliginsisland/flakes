package dialer

import (
    "context"
    "errors"
    "net"
    "strings"

    "golang.org/x/net/proxy"
)

// RewritingDialer wraps a proxy.ContextDialer and rewrites hostnames
// based on a specific pattern before passing them to the underlying dialer.
type RewritingDialer struct {
    Dialer proxy.ContextDialer
    Suffix string
}

// Dial calls DialContext with a background context.
func (rd *RewritingDialer) Dial(network, address string) (net.Conn, error) {
    return rd.DialContext(context.Background(), network, address)
}

// DialContext rewrites the hostname in the address if it matches the pattern
// *.<label>.pacman and then delegates to the underlying dialer.
func (rd *RewritingDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
    host, port, err := net.SplitHostPort(address)
    if err != nil {
        return nil, err
    }

    // Check if the host matches the pattern *.<label>.pacman
    if subdomain, ok := strings.CutSuffix(host, "."+rd.Suffix); ok {
        // Extract the part before the suffix (e.g., "10.0.0.1" from "10.0.0.1.ocna.pacman")
        host = subdomain
        if host == "" {
            return nil, errors.New("Invalid hostname")
        }
        address = net.JoinHostPort(host, port)
    }

    return rd.Dialer.DialContext(ctx, network, address)
}
