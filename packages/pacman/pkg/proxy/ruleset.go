package proxy

import (
	"encoding"
	"errors"
	"net/url"
)

var ErrProxyNotFound = errors.New("proxy not found")

type RuleSet struct {
	Proxies map[string]*URL `json:"proxies"`
	Rules   []*Rule         `json:"rules"`
}

type Rule struct {
	Hosts   []string `json:"hosts"`
	Proxies []string `json:"proxies"`
}

type URL struct {
	url.URL
}

var _ encoding.TextUnmarshaler = (*URL)(nil)

func (p *URL) UnmarshalText(text []byte) error {
	return p.UnmarshalBinary(text)
}
