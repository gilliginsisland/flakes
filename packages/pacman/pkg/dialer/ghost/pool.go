package ghost

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"golang.org/x/net/proxy"
)

// refDialer wraps a ContextDialer with reference counting
type refDialer struct {
	proxy.ContextDialer
	ref chan int
}

// newDialer creates a pooled dialer.
func (g *Dialer) newDialer(u *URL) (*refDialer, error) {
	slog.Debug(
		"Dialer created",
		slog.String("proxy", u.Redacted()),
	)

	d, err := proxy.FromURL(&u.URL, g)
	if err != nil {
		return nil, err
	}

	xd, ok := d.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.Redacted())
	}

	pd := &refDialer{
		ContextDialer: xd,
	}
	// reference counting only applies if the underlying dialer
	// supports being closed.
	if _, ok := d.(io.Closer); ok {
		slog.Debug(
			"Initializing ref counts",
			slog.String("proxy", u.Redacted()),
		)
		pd.ref = make(chan int, 100)
	}
	go g.monitor(u, pd)

	return pd, nil
}

// monitor manages the ref count and removes when inactive.
func (g *Dialer) monitor(u *URL, d *refDialer) {
	var (
		refCount int
		timeout  <-chan time.Time
		wait     chan error
	)

	if w, ok := d.ContextDialer.(interface{ Wait() error }); ok {
		wait = make(chan error, 1)
		go func() {
			wait <- w.Wait()
			close(wait)
		}()
	}

	slog.Debug(
		"Started monitoring dialer",
		slog.String("proxy", u.Redacted()),
	)

loop:
	for {
		select {
		case i := <-d.ref:
			refCount += i
			if refCount > 0 {
				slog.Debug(
					"clearing timeout",
					slog.String("proxy", u.Redacted()),
					slog.Int("refCount", refCount),
				)
				timeout = nil
			} else {
				slog.Debug(
					"Setting timeout",
					slog.String("proxy", u.Redacted()),
					slog.Int("refCount", refCount),
				)
				timeout = time.After(10 * time.Minute)
			}

		case <-timeout:
			slog.Debug(
				"Dialer inactivity timeout reached",
				slog.String("proxy", u.Redacted()),
			)
			timeout = nil
			break loop

		case <-wait:
			slog.Debug(
				"Dialer closed",
				slog.String("proxy", u.Redacted()),
			)
			break loop
		}
	}

	g.dialers.Delete(u)
	slog.Debug(
		"Removed dialer from pool",
		slog.String("proxy", u.Redacted()),
	)

	if c, ok := d.ContextDialer.(io.Closer); ok {
		slog.Debug(
			"Closing dialer",
			slog.String("proxy", u.Redacted()),
		)
		c.Close()
	}
}
