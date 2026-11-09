package oc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
)

type Dialer struct {
	*stackutil.Dialer
	*openconnect.Conn
}

func NewDialer(ctx context.Context, u *url.URL) (_ *Dialer, err error) {
	cb := callbacks{url: u}
	cb.ctx, cb.cancel = context.WithCancelCause(context.WithoutCancel(ctx))
	stop := context.AfterFunc(ctx, func() {
		cb.cancel(context.Cause(ctx))
	})
	defer stop()
	if l, _ := cb.ctx.Value("label").(string); l != "" {
		cb.label = l
	} else {
		cb.label = cb.url.Redacted()
	}

	var (
		conn        *openconnect.Conn
		stackDialer *stackutil.Dialer
		ready       = make(chan struct{})
	)
	defer func() {
		if err == nil {
			return
		}
		if cause := context.Cause(cb.ctx); cause != nil {
			err = cause
		}
		cb.cancel(err)
		if conn != nil {
			conn.Close()
		}
	}()
	browserCtx, browserClose := context.WithCancel(cb.ctx)
	defer browserClose()
	callbacks := openconnect.Callbacks{
		Progress: cb.Progress,
		ProcessAuthForm: (&openconnect.AggregateProcessor{
			openconnect.LoggerFunc(cb.DebugLog), &cb,
		}).ProcessForm,
		ExternalBrowser: func(uri string) error {
			go cb.ExternalBrowser(browserCtx, uri)
			return nil
		},
		CookieObtained: func(string) {
			browserClose()
		},
		ValidatePeerCert: func(cert string) bool {
			return true
		},
		SetupTunHandler: func() error {
			// Native startup delays this callback until initial MTU detection ends.
			dialer, setupErr := withConn(conn)
			if setupErr != nil {
				cb.cancel(setupErr)
			} else {
				stackDialer = dialer
				go func() {
					<-conn.Done()
					dialer.Close()
				}()
			}
			close(ready)
			return setupErr
		},
	}
	if u.Scheme == "anyconnect" {
		callbacks.ProcessCSD = cb.ProcessCSD
	}

	var logLevel openconnect.LogLevel
	switch {
	case slog.Default().Enabled(ctx, slog.LevelDebug):
		logLevel = openconnect.LogLevelTrace
	case slog.Default().Enabled(ctx, slog.LevelInfo):
		logLevel = openconnect.LogLevelInfo
	case slog.Default().Enabled(ctx, slog.LevelError):
		logLevel = openconnect.LogLevelErr
	}

	query := u.Query()
	conn, err = openconnect.Connect(cb.ctx, openconnect.Options{
		Protocol:            openconnect.Protocol(u.Scheme),
		Server:              fmt.Sprintf("%s%s", u.Host, u.Path),
		ForceDPD:            5,
		LogLevel:            logLevel,
		AllowInsecureCrypto: true,
		DeviceIdentity: openconnect.DeviceIdentity{
			PlatformVersion: query.Get("platform-version"),
			DeviceType:      query.Get("device-type"),
			UniqueID:        query.Get("unique-id"),
		},
		Callbacks: callbacks,
	})
	if err != nil {
		return nil, err
	}
	if err = conn.Run(); err != nil {
		return nil, err
	}

	select {
	case <-ready:
	case <-conn.Done():
		cb.cancel(conn.Wait())
	case <-cb.ctx.Done():
	}
	if !stop() {
		cb.cancel(context.Cause(ctx))
	}
	if err = context.Cause(cb.ctx); err != nil {
		return nil, err
	}
	return &Dialer{Dialer: stackDialer, Conn: conn}, nil
}

// Close requests connection shutdown; connection completion closes the stack.
func (d *Dialer) Close() error {
	return d.Conn.Close()
}

// withConn configures netstack from the synchronous native setup-tunnel callback.
// On failure it closes the bridge.
func withConn(conn *openconnect.Conn) (*stackutil.Dialer, error) {
	bridge, ipinfo, err := conn.TunClient()
	if err != nil {
		return nil, err
	}
	ep := &stackutil.Endpoint{Bridge: bridge}
	ep.SetMTU(ipinfo.MTU)
	defer func() {
		if err != nil {
			bridge.Close()
		}
	}()
	opts := stackutil.NetOptions{
		Addr:     ipinfo.Addr,
		Netmask:  ipinfo.Netmask,
		Addr6:    ipinfo.Addr6,
		Netmask6: ipinfo.Netmask6,
		DNS:      ipinfo.DNS,
		Domain:   ipinfo.Domain,
	}
	// Match vpnc-script's fallback when OpenConnect supplies only an IPv6 address.
	if opts.Netmask6 == "" && opts.Addr6 != "" {
		opts.Netmask6 = opts.Addr6 + "/128"
	}
	stackDialer, err := stackutil.NewEndpointDialer(ep, &opts)
	if err != nil {
		return nil, err
	}
	return stackDialer, nil
}
