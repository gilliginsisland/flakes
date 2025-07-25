package ghost

import (
	"encoding"
	"fmt"
	"net/url"
)

type RuleSet []*Rule

type Rule struct {
	Hosts   []string `json:"hosts"`
	Proxies []*Proxy `json:"proxies"`
}

type Proxy struct {
	url.URL
}

var _ encoding.TextUnmarshaler = (*Proxy)(nil)

func (p *Proxy) UnmarshalText(text []byte) error {
	return p.UnmarshalBinary(text)
}

// Principal returns the identity of the URL in the form "user@host",
// or empty string if either part is missing.
func (p *Proxy) Principal() string {
	if p == nil {
		return ""
	}
	user := p.User.Username()
	host := p.Hostname()
	if user == "" || host == "" {
		return ""
	}
	val := user + "@" + host
	return val
}

// ID returns a unique identifier string for the URL pointer.
func (p *Proxy) ID() string {
	return fmt.Sprintf("%p", p)
}
