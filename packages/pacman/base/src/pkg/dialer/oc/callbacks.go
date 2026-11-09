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
