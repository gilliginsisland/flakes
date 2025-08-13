package ghost

import (
	"encoding"
	"errors"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/trie"
)

var ErrProxyNotFound = errors.New("proxy not found")

type RuleSet struct {
	Proxies map[string]*URL `json:"proxies"`
	Rules   []*Rule         `json:"rules"`
}

func (rs RuleSet) Compile() (*trie.Host[[]*URL], error) {
	t := trie.NewHost[[]*URL]()
	for _, r := range rs.Rules {
		urls := make([]*URL, len(r.Proxies))
		for i, proxy := range r.Proxies {
			u, ok := rs.Proxies[proxy]
			if !ok {
				return nil, &url.Error{Op: "compile", URL: proxy, Err: ErrProxyNotFound}
			}
			urls[i] = u
		}
		for _, h := range r.Hosts {
			t.Insert(h, urls)
		}
	}
	return t, nil
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
