package dialer

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"golang.org/x/net/proxy"
)

// waiter defines an interface for dialers that can signal when their connection is lost.
type waiter interface {
	Wait() error
}

// dialer wraps a ContextDialer with reference counting
type dialer struct {
	proxy.ContextDialer
	ref chan int
}

// GetDialer retrieves or creates a pooled dialer.
func (g *GHost) newDialer(u *URL) (*dialer, error) {
	slog.Debug(
		"Dialer created",
		slog.Any("proxy", u),
	)

	d, err := proxy.FromURL(&u.URL, g)
	if err != nil {
		return nil, err
	}

	xd, ok := d.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.String())
	}

	pd := &dialer{
		ContextDialer: xd,
		ref:           make(chan int, 100),
	}
	go g.monitor(u, pd)

	return pd, nil
}

// monitor manages the ref count and removes when inactive.
func (g *GHost) monitor(u *URL, d *dialer) {
	slog.Debug(
		"Started monitoring dialer",
		slog.Any("proxy", u),
	)

	var (
		refCount int
		timeout  <-chan time.Time
		wait     <-chan error
	)

	if w, ok := d.ContextDialer.(waiter); ok {
		ch := make(chan error, 1)
		wait = ch
		go func() {
			slog.Debug(
				"Waiting for dialer to close",
				slog.Any("proxy", u),
			)
			ch <- w.Wait()
			close(ch)
		}()
	}

loop:
	for {
		select {
		case i := <-d.ref:
			refCount += i
			if refCount > 0 {
				timeout = nil
			} else {
				timeout = time.After(30 * time.Second)
			}

		case <-timeout:
			slog.Debug(
				"Dialer inactivity timeout reached",
				slog.Any("proxy", u),
			)
			timeout = nil
			break loop

		case <-wait:
			slog.Debug(
				"Dialer closed",
				slog.Any("proxy", u),
			)
			break loop
		}
	}

	g.dialers.Delete(u)
	slog.Debug(
		"Removed dialer from pool",
		slog.Any("proxy", u),
	)

	if c, ok := d.ContextDialer.(io.Closer); ok {
		slog.Debug(
			"Closing dialer",
			slog.Any("proxy", u),
		)
		c.Close()
	}
}
