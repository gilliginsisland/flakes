package ghost

import (
	"encoding"
	"fmt"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/matcher"
)

type URL struct {
	url.URL
}

var _ encoding.TextUnmarshaler = (*URL)(nil)

func (u *URL) UnmarshalText(text []byte) error {
	return u.UnmarshalBinary(text)
}

// Principal returns the identity of the URL in the form "user@host",
// or empty string if either part is missing.
func (u *URL) Principal() string {
	if u == nil {
		return ""
	}
	user := u.User.Username()
	host := u.Hostname()
	if user == "" || host == "" {
		return ""
	}
	val := user + "@" + host
	return val
}

// ID returns a unique identifier string for the URL pointer.
func (u *URL) ID() string {
	return fmt.Sprintf("%p", u)
}

type HostMatcher struct {
	matcher.StringMatcher
}

var _ encoding.TextUnmarshaler = (*HostMatcher)(nil)

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
