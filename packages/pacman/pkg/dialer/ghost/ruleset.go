package ghost

import (
	"encoding"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/trie"
)

type RuleSet []*Rule

func (rs RuleSet) Compile() *trie.Host[[]*URL] {
	t := trie.NewHost[[]*URL]()
	for _, r := range rs {
		for _, h := range r.Hosts {
			t.Insert(h, r.Proxies)
		}
	}
	return t
}

type Rule struct {
	Hosts   []string `json:"hosts"`
	Proxies []*URL   `json:"proxies"`
}

type URL struct {
	url.URL
}

var _ encoding.TextUnmarshaler = (*URL)(nil)

func (p *URL) UnmarshalText(text []byte) error {
	return p.UnmarshalBinary(text)
}
