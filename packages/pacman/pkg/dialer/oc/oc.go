package oc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/caseymrm/menuet"

	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/openconnect"
	"github.com/gilliginsisland/pacman/pkg/stackutil"
)

var errUserCancelled = errors.New("User cancelled")

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

type Dialer struct {
	*stackutil.Dialer
	*openconnect.Conn
}

func NewDialer(u *url.URL) (*Dialer, error) {
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
		ForceDPD: 5,
		LogLevel: openconnect.LogLevelDebug,
		Callbacks: openconnect.Callbacks{
			Progress: cb.Progress,
			ProcessAuthForm: func(form openconnect.AuthForm) openconnect.FormResult {
				err := processAuthForm(form, u)
				if err != nil {
					slog.Error(
						"openconnect form authentication failed",
						slog.String("proxy", u.Redacted()),
						slog.Any("error", err),
					)
					if errors.Is(err, errUserCancelled) {
						return openconnect.FormResultCancelled
					}
					return openconnect.FormResultErr
				}
				slog.Debug(
					"form authentication succeeded",
					slog.String("proxy", u.Redacted()),
				)
				return openconnect.FormResultOk
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

func processAuthForm(form openconnect.AuthForm, u *url.URL) error {
	app := menuet.App()

	slog.Debug(
		"Processing Auth Form",
		slog.String("banner", form.Banner),
		slog.String("message", form.Message),
		slog.String("error", form.Error),
	)

	if form.Error != "" {
		app.Alert(menuet.Alert{
			MessageText:     "Authentication Error: " + u.Redacted(),
			InformativeText: form.Error,
		})
	}

	for _, opt := range form.Options {
		slog.Debug(
			"option",
			slog.String("name", opt.Name),
			slog.String("label", opt.Label),
			slog.String("type", opt.Type.String()),
		)
		switch {
		case opt.Type == openconnect.FormOptionText && strings.HasPrefix(strings.ToLower(opt.Name), "user"):
			opt.SetValue(u.User.Username())
		case opt.Type == openconnect.FormOptionPassword:
			passwd, _ := u.User.Password()
			if u.Query().Get("token") == "otp" {
				response := <-notify.Notify(notify.Notification{
					Title:               "Authentication Required",
					Message:             fmt.Sprintf("OTP is required for the proxy at %s", u.Redacted()),
					ResponsePlaceholder: "YubiKey OTP",
				})
				if response == "" {
					return errUserCancelled
				} else {
					passwd += response
				}
			}
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

	return nil
}
