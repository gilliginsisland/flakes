package matcher

import (
	"net"
	"regexp"
)

var cidrRegex = regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2}$|^[0-9a-fA-F:]+/[0-9]{1,3}$`)

// CIDRMatcher holds a parsed IP network.
type CIDRMatcher struct {
	network *net.IPNet
}

// NewCIDRMatcher parses the CIDR string and returns a CIDRMatcher.
func NewCIDRMatcher(cidr string) (StringMatcher, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return &CIDRMatcher{network: network}, nil
}

// MatchString checks if the provided string IP is within the CIDR range.
func (c *CIDRMatcher) MatchString(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return c.network.Contains(ip)
}
