package app

import (
	"context"
	"errors"
	"net/url"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/authenticationservices"
)

// Browser keeps an interactive authentication session open until ctx is cancelled.
// Earlier completion is an error; cancellation returns ctx.Err().
func (pacman *PACMan) Browser(ctx context.Context, uri string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u, err := url.Parse(uri)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" {
		return errors.New("invalid browser authentication URL")
	}
	anchor := authenticationservices.NewAnchorWindow(appkit.WindowOptions{
		ContentRect: appkit.Rect{Width: 1, Height: 1},
		Backing:     appkit.BackingStoreBuffered,
	})
	defer anchor.Close()
	anchor.SetRestorable(false).SetOpaque(false).SetAlphaValue(0).SetIgnoresMouseEvents(true)
	completed := make(chan error, 1)
	session := authenticationservices.NewSession(uri, "", func(_ string, err error) {
		select {
		case completed <- err:
		default:
		}
	})
	if session == nil {
		return errors.New("create browser authentication session")
	}
	defer session.Cancel()
	anchor.WithPointer(func(pointer unsafe.Pointer) { session.SetPresentationContextProvider(pointer) })
	session.SetPrefersEphemeralWebBrowserSession(false)
	anchor.MakeKeyAndOrderFront()
	if !session.Start() {
		return errors.New("start browser authentication session")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-completed:
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			return errors.New("browser closed before VPN authentication completed")
		}
		return err
	}
}
