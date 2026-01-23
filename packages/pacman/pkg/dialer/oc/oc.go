package oc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

var NotificationCategories = []menuet.NotificationCategory{
	{
		Identifier: "yubi-auth",
		Actions: []menuet.Actioner{
			menuet.NotificationActionText{
				NotificationAction: menuet.NotificationAction{
					Identifier: "yubi-auth-token",
					Title:      "Reply",
				},
				TextInputButtonTitle: "Submit",
				TextInputPlaceholder: "Enter YubiKey OTP",
			},
		},
		Options: menuet.CategoryOptionCustomDismiss,
	},
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

func (cb *callbacks) ProcessForm(form *openconnect.AuthForm) openconnect.FormResult {
	if form.Error != "" {
		notify.Notify(notify.Notification{
			Title:    cb.label,
			Subtitle: "Authentication Error",
			Body:     form.Error,
		})
		return openconnect.FormResultErr
	}

	cb.cp.Username = cb.url.User.Username()
	cb.cp.Password, _ = cb.url.User.Password()

	if cb.url.Query().Get("token") == "otp" {
		resp, err := notify.NotifyCtx(cb.ctx, notify.Notification{
			CategoryIdentifier: "yubi-auth",
			Title:              cb.label,
			Subtitle:           "Authentication Required",
			Body:               "Enter YubiKey OTP",
		})
		if err != nil {
			cb.DebugLog("AuthForm ctx cancelled", slog.Any("error", err))
			return openconnect.FormResultCancelled
		}
		switch resp.ActionIdentifier {
		case menuet.DismissActionIdentifier:
			cb.DebugLog("Auth form user cancelled")
			return openconnect.FormResultCancelled
		case menuet.DefaultActionIdentifier:
			resp, err := menuet.DisplayCtx(cb.ctx, menuet.Alert{
				MessageText:     "YubiKey OTP Authentication Required",
				InformativeText: fmt.Sprintf("The proxy %s requires YubiKey OTP", cb.label),
				Inputs: []string{
					"Enter YubiKey OTP",
				},
				Buttons: []string{
					"Submit",
					"Cancel",
				},
			})
			if err != nil || resp.Button == 1 {
				cb.DebugLog("AuthForm ctx cancelled", slog.Any("error", err))
				return openconnect.FormResultCancelled
			}
			cb.cp.Password += resp.Inputs[0]
		case "yubi-auth-token":
			cb.cp.Password += resp.Text
		}
		cb.DebugLog("AuthForm YOTP received")
	}

	result := cb.cp.ProcessForm(form)
	cb.DebugLog("CredentialsProcessor", slog.String("result", result.String()))
	return result
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

	var csd string
	if csd = u.Query().Get("csd"); csd == "" {
		switch u.Scheme {
		case "anyconnect":
			csd, _ = os.Executable()
		}
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
		CSD:                 csd,
		ForceDPD:            5,
		LogLevel:            logLevel,
		AllowInsecureCrypto: true,
		Callbacks: openconnect.Callbacks{
			Progress: cb.Progress,
			ProcessAuthForm: (&openconnect.AggregateProcessor{
				openconnect.LoggerFunc(cb.DebugLog), &cb,
			}).ProcessForm,
			ExternalBrowser: cb.ExternalBrowser,
			ValidatePeerCert: func(cert string) bool {
				return true
			},
		},
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
		defer rwc.Close()
		conn.Wait()
	}()

	return &Dialer{
		Conn:   conn,
		Dialer: d,
	}, nil
}
