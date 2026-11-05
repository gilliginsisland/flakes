package oc

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/openconnect/hostscan"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
)

type callbacks struct {
	url    *url.URL
	ctx    context.Context
	cp     openconnect.CredentialsProcessor
	label  string
	cancel context.CancelCauseFunc
}

func (cb *callbacks) Progress(level openconnect.LogLevel, message string) {
	var l slog.Level
	switch level {
	case openconnect.LogLevelErr:
		l = slog.LevelError
	case openconnect.LogLevelInfo:
		l = slog.LevelInfo
	case openconnect.LogLevelDebug, openconnect.LogLevelTrace:
		l = slog.LevelDebug
	default:
		l = slog.LevelDebug
	}
	slog.Log(
		context.Background(), l, message,
		slog.String("proxy", cb.label),
	)
}

func (cb *callbacks) DebugLog(msg string, xtras ...slog.Attr) {
	xtras = append(xtras, slog.String("proxy", cb.label))
	slog.LogAttrs(context.Background(), slog.LevelDebug, msg, xtras...)
}

func (cb *callbacks) ExternalBrowser(ctx context.Context, uri string) (err error) {
	defer func() {
		// Cookie acquisition cancels the browser context when authentication finishes.
		// That expected closure must not cancel the connection's context.
		if err != nil && !errors.Is(err, ctx.Err()) {
			cb.cancel(err)
		}
	}()
	if DefaultBrowserProvider == nil {
		return errors.New("browser authentication provider is not configured")
	}
	return DefaultBrowserProvider.Browser(ctx, uri)
}

func (cb *callbacks) ProcessForm(form *openconnect.AuthForm) error {
	if form.Error != "" {
		return errors.New(form.Error)
	}

	cb.cp.Username = cb.url.User.Username()
	if DefaultInputProvider == nil {
		return errors.New("authentication input provider is not configured")
	}
	password, err := DefaultInputProvider.Input(cb.ctx, cb.url, InputPassword)
	if err != nil {
		cb.DebugLog("Auth form input failed", slog.Any("error", err))
		return err
	}
	cb.cp.Password = password

	err = cb.cp.ProcessForm(form)
	if err != nil {
		cb.DebugLog("CredentialsProcessor", slog.Any("error", err))
		return err
	}
	cb.DebugLog("CredentialsProcessor", slog.String("result", "ok"))
	return nil
}

func (cb *callbacks) ProcessCSD(info openconnect.CSDInfo) error {
	cb.DebugLog(
		"Processing CSD",
		slog.String("hostname", info.Hostname),
		slog.String("ticket", info.Ticket),
		slog.String("stub", info.Stub),
	)

	hash, err := base64.StdEncoding.DecodeString(info.SHA256)
	if err != nil {
		return fmt.Errorf("invalid CSD SHA256: %w", err)
	}

	// Native authentication waits synchronously for this callback. Its HTTP
	// requests must observe setup cancellation so Reset can finish the factory.
	client := hostscan.NewClient(info.Hostname, hash)
	token := info.Token
	if info.Ticket != "" {
		stub := info.Stub
		if stub == "" {
			stub = "0"
		}
		token, err = client.GetToken(cb.ctx, info.Ticket, stub)
		if err != nil {
			return fmt.Errorf("fetch hostscan token: %w", err)
		}
	}

	manifest, err := client.GetManifest(cb.ctx)
	if err != nil {
		return fmt.Errorf("get hostscan policy: %w", err)
	}

	res, err := client.PostReport(cb.ctx, hostscan.NewMockScanner(manifest).Scan(), token)
	if err != nil {
		return fmt.Errorf("post hostscan report: %w", err)
	}
	defer res.Body.Close()

	txt, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read hostscan response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("hostscan server returned error: %s", string(txt))
	}

	cb.DebugLog("CSD completed successfully", slog.String("response", string(txt)))
	return nil
}

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
		conn  *openconnect.Conn
		d     *Dialer
		ready = make(chan struct{})
	)
	defer func() {
		if err == nil {
			return
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
			var err error
			if d, err = WithConn(conn); err != nil {
				cb.cancel(err)
			}
			close(ready)
			return err
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
		if cause := context.Cause(cb.ctx); cause != nil {
			err = cause
		}
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
	return d, nil
}

// WithConn configures netstack from the synchronous native setup-tunnel callback.
func WithConn(conn *openconnect.Conn) (*Dialer, error) {
	rwc, ipinfo, err := conn.TunClient()
	if err != nil {
		return nil, err
	}

	opts := stackutil.NetOptions{
		Addr:     ipinfo.Addr,
		Netmask:  ipinfo.Netmask,
		Addr6:    ipinfo.Addr6,
		Netmask6: ipinfo.Netmask6,
		DNS:      ipinfo.DNS,
		Domain:   ipinfo.Domain,
		MTU:      ipinfo.MTU,
	}
	// Match vpnc-script's fallback when OpenConnect supplies only an IPv6 address.
	if opts.Netmask6 == "" && opts.Addr6 != "" {
		opts.Netmask6 = opts.Addr6 + "/128"
	}
	d, err := stackutil.NewTunDialer(rwc, &opts)
	if err != nil {
		return nil, err
	}

	return &Dialer{
		Conn:   conn,
		Dialer: d,
	}, nil
}
