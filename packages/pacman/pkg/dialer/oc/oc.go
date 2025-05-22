package oc

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"

	"github.com/gilliginsisland/pacman/internal/stackutil"
	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"golang.org/x/net/proxy"
)

type callbacks struct {
	url *url.URL
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

type dialer struct {
	*stackutil.Dialer
	*openconnect.Conn
}

func New(u *url.URL, _ proxy.Dialer) (proxy.Dialer, error) {
	cb := callbacks{url: u}

	var csd string
	switch u.Scheme {
	case "anyconnect":
		csd, _ = os.Executable()
	}

	conn, err := openconnect.Connect(context.Background(), openconnect.Options{
		Protocol: openconnect.Protocol(u.Scheme),
		Server:   fmt.Sprintf("%s%s", u.Host, u.Path),
		CSD:      csd,
		LogLevel: openconnect.LogLevelDebug,
		Callbacks: openconnect.Callbacks{
			Progress: cb.Progress,
			ProcessAuthForm: func(form openconnect.AuthForm) openconnect.FormResult {
				return processAuthForm(form, u.User)
			},
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

func WithConn(conn *openconnect.Conn) (proxy.Dialer, error) {
	rwc, ipinfo, err := conn.TunClient()
	if err != nil {
		return nil, err
	}

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

	go func() {
		defer rwc.Close()
		conn.Run()
	}()

	return &dialer{
		Conn:   conn,
		Dialer: d,
	}, nil
}

func processAuthForm(form openconnect.AuthForm, user *url.Userinfo) openconnect.FormResult {
	slog.Debug(
		"Processing Auth Form",
		slog.String("banner", form.Banner),
		slog.String("message", form.Message),
		slog.String("error", form.Error),
	)

	for _, opt := range form.Options {
		slog.Debug(
			"option",
			slog.String("name", opt.Name),
			slog.String("label", opt.Label),
			slog.String("type", opt.Type.String()),
		)
		switch opt.Name {
		case "user":
			opt.SetValue(user.Username())
		case "passwd":
			passwd, _ := user.Password()
			opt.SetValue(passwd)
		}
		for _, choice := range opt.Choices {
			slog.Debug(
				"choice",
				slog.String("name", choice.Name),
				slog.String("label", choice.Label),
			)
		}
	}

	return openconnect.FormResultOk
}
