package dialer

import (
	"context"
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
)

func init() {
	RegisterContextDialerType("anyconnect", Openconnect)
	RegisterContextDialerType("gp", Openconnect)
}

func Openconnect(ctx context.Context, u *url.URL, fwd proxy.Dialer) (Dialer, error) {
	d, err := oc.NewDialer(ctx, u)
	if err != nil {
		return nil, err
	}
	return d, nil
}
