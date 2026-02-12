package dialer

import (
	"context"
	"io"
	"net/url"

	"golang.org/x/net/proxy"
)

// ctxSchemes is a map from URL schemes to a function that creates a Dialer
// from a URL with such a scheme.
var ctxSchemes map[string]func(context.Context, *url.URL, proxy.Dialer) (proxy.Dialer, error)

// RegisterContextDialerType takes a URL scheme and a function to generate Dialers from
// a URL with that scheme and a forwarding Dialer. Registered schemes are used
// by FromURL.
func RegisterContextDialerType(scheme string, fn func(context.Context, *url.URL, proxy.Dialer) (proxy.Dialer, error)) {
	if ctxSchemes == nil {
		ctxSchemes = make(map[string]func(context.Context, *url.URL, proxy.Dialer) (proxy.Dialer, error))
	}
	ctxSchemes[scheme] = fn

	// Register background-compatible version with the standard proxy registry.
	proxy.RegisterDialerType(scheme, func(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
		return fn(context.Background(), u, forward)
	})
}

// FromURLContext behaves like proxy.FromURL but supports context and cancellation.
//
// If the scheme was registered with RegisterContextDialerType, it uses that.
// Otherwise, it falls back to proxy.FromURL in a goroutine so that
// ctx cancellation can return immediately even if setup is blocking.
func FromURLContext(ctx context.Context, u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	if fn := ctxSchemes[u.Scheme]; fn != nil {
		return fn(ctx, u, forward)
	}
	var (
		dialer proxy.Dialer
		err    error
		done   = make(chan struct{}, 1)
	)
	go func() {
		dialer, err = proxy.FromURL(u, forward)
		close(done)
	}()
	select {
	case <-ctx.Done():
		go func() {
			<-done
			if c, ok := dialer.(io.Closer); ok {
				c.Close()
			}
		}()
		return nil, ctx.Err()
	case <-done:
		return dialer, err
	}
}
