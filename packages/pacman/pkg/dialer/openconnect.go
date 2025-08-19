package dialer

import (
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
)

func init() {
	proxy.RegisterDialerType("anyconnect", oc.FromURL)
	proxy.RegisterDialerType("gp", oc.FromURL)
}
