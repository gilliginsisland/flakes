package pacman

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"regexp"

	"golang.org/x/net/proxy"
)

// Works like DialContext on net.Dialer but using the passed dialer.
//
// The passed ctx is only used for returning the Conn, not the lifetime of the Conn.
//
// Dialers that do not implement ContextDialer can leak a goroutine for as long as it
// takes the underlying Dialer implementation to timeout.
//
// A Conn returned from a successful Dial after the context has been cancelled will be immediately closed.
func dial(ctx context.Context, d proxy.Dialer, network, address string) (net.Conn, error) {
	if ctx == nil {
		return d.Dial(network, address)
	}

	if xd, ok := d.(proxy.ContextDialer); ok {
		return xd.DialContext(ctx, network, address)
	}

	var (
		conn net.Conn
		done = make(chan net.Conn, 1)
		err  error
	)
	go func() {
		conn, err = d.Dial(network, address)
		close(done)
		if conn != nil && ctx.Err() != nil {
			conn.Close()
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-done:
	}
	return conn, err
}

type Rule struct {
	matcher *regexp.Regexp
	dialers []proxy.Dialer
}

// Dialer directs connections based on rules.
// It supports recursive dialers.
type Dialer struct {
	Rules []*Rule
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(nil, network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialers, err := d.dialersForAddress(address)
	if err != nil {
		return nil, err
	}

	for _, d := range dialers {
		conn, err := dial(ctx, d, network, address)
		if err != nil {
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("all dialers failed for %s", address)
}

func (d *Dialer) dialersForAddress(address string) ([]proxy.Dialer, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	for _, r := range d.Rules {
		if r.matcher.MatchString(host) {
			return r.dialers, nil
		}
	}

	return []proxy.Dialer{proxy.Direct}, nil
}

func (d *Dialer) loadRawRule(host string, proxies ...string) error {
	var rule Rule

	if re, err := compileWildcard(host); err != nil {
		return fmt.Errorf("invalid host matcher '%s': %w", host, err)
	} else {
		rule.matcher = re
	}

	for _, str := range proxies {
		u, err := url.Parse(str)
		if err != nil {
			return fmt.Errorf("invalid proxy URL '%s': %w", str, err)
		}

		p, err := proxy.FromURL(u, d)
		if err != nil {
			slog.Warn(fmt.Sprintf("skipping unsupported proxy: %s: host: %s: %s", u, host, err.Error()))
		}

		rule.dialers = append(rule.dialers, p)
	}

	if l := len(rule.dialers); l == 0 {
		return fmt.Errorf("All proxies are invalid")
	}

	d.Rules = append(d.Rules, &rule)
	return nil
}

func (d *Dialer) LoadRulesFile(r io.Reader) error {
	var rules []struct {
		Host    string   `json:"host"`
		Proxies []string `json:"proxies"`
	}

	if err := json.NewDecoder(r).Decode(&rules); err != nil {
		return err
	}

	for _, raw := range rules {
		err := d.loadRawRule(raw.Host, raw.Proxies...)
		if err != nil {
			return err
		}
	}

	return nil
}
