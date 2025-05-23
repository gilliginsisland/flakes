package ghost

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/gilliginsisland/pacman/pkg/notify"
	"golang.org/x/net/proxy"
)

// refDialer wraps a ContextDialer with reference counting
type refDialer struct {
	proxy.ContextDialer
	ref chan int
}

// newDialer creates a pooled dialer.
func (d *Dialer) newDialer(u *URL) (*refDialer, error) {
	slog.Debug(
		"Dialer created",
		slog.String("proxy", u.Redacted()),
	)

	d.notifier.Send(notify.Notification{
		Subtitle: "Connecting proxy",
		Message:  u.Redacted(),
	})

	dd, err := proxy.FromURL(&u.URL, d)
	if err != nil {
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.Redacted())
	}

	pd := &refDialer{
		ContextDialer: xd,
	}
	// reference counting only applies if the underlying dialer
	// supports being closed.
	if _, ok := dd.(io.Closer); ok {
		slog.Debug(
			"Initializing ref counts",
			slog.String("proxy", u.Redacted()),
		)
		pd.ref = make(chan int, 100)
	}
	go d.monitor(u, pd)

	d.notifier.Send(notify.Notification{
		Subtitle: "Proxy connected",
		Message:  u.Redacted(),
	})

	return pd, nil
}

// monitor manages the ref count and removes when inactive.
func (d *Dialer) monitor(u *URL, rd *refDialer) {
	var (
		refCount int
		timeout  <-chan time.Time
		wait     chan error
	)

	if w, ok := rd.ContextDialer.(interface{ Wait() error }); ok {
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
		case i := <-rd.ref:
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
			d.notifier.Send(notify.Notification{
				Subtitle: "Proxy disconnected",
				Message:  u.Redacted(),
			})
			break loop
		}
	}

	d.pool.Delete(u)
	slog.Debug(
		"Removed dialer from pool",
		slog.String("proxy", u.Redacted()),
	)

	if c, ok := rd.ContextDialer.(io.Closer); ok {
		slog.Debug(
			"Closing dialer",
			slog.String("proxy", u.Redacted()),
		)
		c.Close()
	}
}
