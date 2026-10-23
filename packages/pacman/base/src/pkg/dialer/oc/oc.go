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

	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/openconnect/hostscan"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

var NotificationCategories = []menuet.NotificationCategory{
	{
		Identifier: "external-browser-auth",
		Actions: []menuet.Actioner{
			menuet.NotificationAction{
				Identifier: "external-browser-auth-open",
				Title:      "Open",
			},
		},
		Options: menuet.CategoryOptionCustomDismiss,
	},
}

type callbacks struct {
	url   *url.URL
	ctx   context.Context
	cp    openconnect.CredentialsProcessor
	label string
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

func (cb *callbacks) ExternalBrowser(uri string) error {
	resp, err := notify.NotifyCtx(cb.ctx, notify.Notification{
		CategoryIdentifier: "external-browser-auth",
		Title:              cb.label,
		Subtitle:           "Authentication Required",
		Body:               "Click to complete authentication in browser",
		InterruptionLevel:  menuet.NotificationInterruptionLevelTimeSensitive,
	})
	if err != nil {
		cb.DebugLog("ExternalBrowser ctx cancelled", slog.Any("error", err))
		return err
	}
	if resp.ActionIdentifier == menuet.DismissActionIdentifier {
		err = errors.New("ExternalBrowser user cancelled")
		cb.DebugLog("ExternalBrowser user cancelled", slog.Any("error", err))
		return err
	}
	return xdg.Run(uri)
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
	cb.DebugLog("Processing CSD",
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

func NewDialer(ctx context.Context, u *url.URL) (*Dialer, error) {
	cb := callbacks{
		url: u,
		ctx: ctx,
	}
	if l, _ := cb.ctx.Value("label").(string); l != "" {
		cb.label = l
	} else {
		cb.label = cb.url.Redacted()
	}

	callbacks := openconnect.Callbacks{
		Progress: cb.Progress,
		ProcessAuthForm: (&openconnect.AggregateProcessor{
			openconnect.LoggerFunc(cb.DebugLog), &cb,
		}).ProcessForm,
		ExternalBrowser: cb.ExternalBrowser,
		ValidatePeerCert: func(cert string) bool {
			return true
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

	conn, err := openconnect.Connect(ctx, openconnect.Options{
		Protocol:            openconnect.Protocol(u.Scheme),
		Server:              fmt.Sprintf("%s%s", u.Host, u.Path),
		ForceDPD:            5,
		LogLevel:            logLevel,
		AllowInsecureCrypto: true,
		Callbacks:           callbacks,
	})
	if err != nil {
		return nil, err
	}

	d, err := WithConn(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return d, nil
}

func WithConn(conn *openconnect.Conn) (*Dialer, error) {
	rwc, ipinfo, err := conn.TunClient()
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			rwc.Close()
		}
	}()

	d, err := stackutil.NewTunDialer(rwc, &stackutil.NetOptions{
		Addr:     ipinfo.Addr,
		Netmask:  ipinfo.Netmask,
		Addr6:    ipinfo.Addr6,
		Netmask6: ipinfo.Netmask6,
		DNS:      ipinfo.DNS,
		Domain:   ipinfo.Domain,
		MTU:      ipinfo.MTU,
	})
	if err != nil {
		return nil, err
	}

	err = conn.Run()
	if err != nil {
		return nil, err
	}

	go func() {
		// Conn owns VPN lifetime. Closing rwc after death ends the packet-copy
		// workers, whose cleanup closes netstack and unblocks its connections.
		defer rwc.Close()
		conn.Wait()
	}()

	return &Dialer{
		Conn:   conn,
		Dialer: d,
	}, nil
}
