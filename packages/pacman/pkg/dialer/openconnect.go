package dialer

import (
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
)

func init() {
	proxy.RegisterDialerType("anyconnect", oc.New)
	proxy.RegisterDialerType("gp", oc.New)
}
