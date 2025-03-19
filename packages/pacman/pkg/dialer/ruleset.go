package dialer

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/gilliginsisland/pacman/pkg/matcher"
	"golang.org/x/net/proxy"
)

type Rule struct {
	matchers []matcher.StringMatcher
	dialers  []proxy.Dialer
}

type Ruleset struct {
	rules   []*Rule
	factory *Factory
}

// NewRuleset creates a new Ruleset with a fallback dialer.
func NewRuleset(fwd proxy.Dialer) *Ruleset {
	return &Ruleset{
		factory: NewFactory(fwd),
	}
}

func (rs *Ruleset) loadRawRule(hosts []string, proxies []string) error {
	var rule Rule

	for _, h := range hosts {
		m, err := matcher.CompileMatcher(h)
		if err != nil {
			slog.Warn(fmt.Sprintf("skipping invalid host matcher '%s': %s", h, err.Error()))
			continue
		}
		rule.matchers = append(rule.matchers, m)
	}

	if l := len(rule.matchers); l == 0 {
		return fmt.Errorf("All hosts are invalid")
	}

	for _, p := range proxies {
		d, err := rs.factory.Get(p)
		if err != nil {
			slog.Warn(fmt.Sprintf("skipping unsupported proxy: %s: %s", p, err.Error()))
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
		Hosts   []string `json:"hosts"`
		Proxies []string `json:"proxies"`
	}

	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}

	for _, raw := range rules {
		err := rs.loadRawRule(raw.Hosts, raw.Proxies)
		if err != nil {
			return err
		}
	}

	return nil
}
