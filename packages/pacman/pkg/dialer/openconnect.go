package dialer

import (
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
)

func init() {
	proxy.RegisterDialerType("anyconnect", Openconnect)
	proxy.RegisterDialerType("gp", Openconnect)
}

func Openconnect(u *url.URL, fwd proxy.Dialer) (proxy.Dialer, error) {
	return oc.NewDialer(u)
}
