package oc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"

	"github.com/caseymrm/menuet"

	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
)

type callbacks struct {
	url *url.URL
	ctx context.Context
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
		slog.String("proxy", cb.url.Redacted()),
	)
}

func (cb *callbacks) DebugLog(msg string, xtras ...slog.Attr) {
	xtras = append(xtras, slog.String("proxy", cb.url.Redacted()))
	slog.LogAttrs(context.Background(), slog.LevelDebug, msg, xtras...)
}

func (cb *callbacks) ProcessForm(form *openconnect.AuthForm) openconnect.FormResult {
	app := menuet.App()

	if form.Error != "" {
		app.Alert(menuet.Alert{
			MessageText:     "Authentication Error: " + cb.url.Redacted(),
			InformativeText: form.Error,
		})
	}

	passwd, _ := cb.url.User.Password()
	if cb.url.Query().Get("token") == "otp" {
		notif := notify.Notification{
			Title:               "Authentication Required",
			Message:             fmt.Sprintf("OTP is required for the proxy at %s", cb.url.Redacted()),
			ResponsePlaceholder: "YubiKey OTP",
		}
		select {
		case response := <-notify.Notify(notif):
			if response == "" {
				cb.DebugLog("Auth form user cancelled")
				return openconnect.FormResultCancelled
			}
			passwd += response
			cb.DebugLog("AuthForm YOTP received")
		case <-cb.ctx.Done():
			cb.DebugLog("AuthForm ctx cancelled")
			return openconnect.FormResultCancelled
		}
	}

	result := (&openconnect.CredentialsProcessor{
		Username: cb.url.User.Username(),
		Password: passwd,
	}).ProcessForm(form)
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

	var csd string
	switch u.Scheme {
	case "anyconnect":
		csd, _ = os.Executable()
	}

	conn, err := openconnect.Connect(ctx, openconnect.Options{
		Protocol: openconnect.Protocol(u.Scheme),
		Server:   fmt.Sprintf("%s%s", u.Host, u.Path),
		CSD:      csd,
		ForceDPD: 5,
		LogLevel: openconnect.LogLevelDebug,
		Callbacks: openconnect.Callbacks{
			Progress: cb.Progress,
			ProcessAuthForm: (&openconnect.AggregateProcessor{
				openconnect.LoggerFunc(cb.DebugLog),
				&cb,
			}).ProcessForm,
			ExternalBrowser: func(uri string) error {
				return exec.Command("open", uri).Run()
			},
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
