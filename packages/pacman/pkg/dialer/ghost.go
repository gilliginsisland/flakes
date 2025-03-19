package dialer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"

	"github.com/gilliginsisland/pacman/internal/netutil"
	"github.com/gilliginsisland/pacman/pkg/matcher"
	"golang.org/x/net/proxy"
)

// GHost directs connections based on rules.
// It supports recursive dialers.
type GHost struct {
	Ruleset *Ruleset
}

func NewGHost() *GHost {
	g := &GHost{}
	g.Ruleset = NewRuleset(g)
	return g
}

func (g *GHost) Dial(network, address string) (net.Conn, error) {
	return g.DialContext(nil, network, address)
}

func (g *GHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialers, err := g.dialersForAddress(address)
	if err != nil {
		return nil, err
	}

	for _, d := range dialers {
		conn, err := netutil.DialContext(ctx, d, network, address)
		if err != nil {
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("all dialers failed for %s", address)
}

func (g *GHost) dialersForAddress(address string) ([]proxy.Dialer, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	for _, r := range g.Ruleset.rules {
		if r.matcher.MatchString(host) {
			return r.dialers, nil
		}
	}

	return []proxy.Dialer{proxy.Direct}, nil
}

type Rule struct {
	matcher matcher.StringMatcher
	dialers []proxy.Dialer
}

type Ruleset struct {
	rules   []*Rule
	fwd     proxy.Dialer
	dialers map[string]proxy.Dialer
}

// NewRuleset creates a new Ruleset with a fallback dialer.
func NewRuleset(fwd proxy.Dialer) *Ruleset {
	return &Ruleset{
		fwd:     fwd,
		dialers: make(map[string]proxy.Dialer),
	}
}

func (rs *Ruleset) dialerFromURL(p string) (proxy.Dialer, error) {
	if d, ok := rs.dialers[p]; ok {
		return d, nil
	}

	u, err := url.Parse(p)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL '%s': %w", p, err)
	}

	d, err := proxy.FromURL(u, rs.fwd)
	if err != nil {
		return nil, err
	}

	rs.dialers[p] = d
	return d, nil
}

func (rs *Ruleset) loadRawRule(host string, proxies []string) error {
	var rule Rule

	if m, err := matcher.CompileMatcher(host); err != nil {
		return fmt.Errorf("invalid host matcher '%s': %w", host, err)
	} else {
		rule.matcher = m
	}

	for _, p := range proxies {
		d, err := rs.dialerFromURL(p)
		if err != nil {
			slog.Warn(fmt.Sprintf("skipping unsupported proxy: %s: host: %s: %s", p, host, err.Error()))
			continue
		}
		rule.dialers = append(rule.dialers, d)
	}

	if l := len(rule.dialers); l == 0 {
		return fmt.Errorf("All proxies are invalid")
	}

	rs.rules = append(rs.rules, &rule)
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (rs *Ruleset) UnmarshalJSON(data []byte) error {
	var rules []struct {
		Host    string   `json:"host"`
		Proxies []string `json:"proxies"`
	}

	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	for _, raw := range rules {
		err := rs.loadRawRule(raw.Host, raw.Proxies)
		if err != nil {
			return err
		}
	}

	return nil
}
