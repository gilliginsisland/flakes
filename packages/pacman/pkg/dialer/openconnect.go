package dialer

import (
	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("anyconnect", oc.New)
	proxy.RegisterDialerType("gp", oc.New)
}
