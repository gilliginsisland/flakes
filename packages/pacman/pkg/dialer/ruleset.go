package dialer

import (
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/matcher"
)

type URL struct {
	url.URL
}

func (u *URL) UnmarshalText(text []byte) error {
	return u.UnmarshalBinary(text)
}

type HostMatcher struct {
	matcher.StringMatcher
}

func (m *HostMatcher) UnmarshalText(text []byte) (err error) {
	m.StringMatcher, err = matcher.CompileMatcher(string(text))
	return err
}

type Rule struct {
	Hosts   []HostMatcher `json:"hosts"`
	Proxies []*URL        `json:"proxies"`
}

type Ruleset []*Rule

func (rs Ruleset) MatchHost(host string) *Rule {
	for _, r := range rs {
		for _, m := range r.Hosts {
			if m.MatchString(host) {
				return r
			}
		}
	}
	return nil
}
