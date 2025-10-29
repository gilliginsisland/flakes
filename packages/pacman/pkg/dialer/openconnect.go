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

func Openconnect(ctx context.Context, u *url.URL, fwd proxy.Dialer) (proxy.Dialer, error) {
	return oc.NewDialer(ctx, u)
}
