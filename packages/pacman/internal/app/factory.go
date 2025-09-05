package app

import (
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
)

type PooledDialer struct {
	Label string
	URL   *url.URL
	Fwd   proxy.Dialer
	lazy  *dialer.Lazy
	node  *MenuNode
	child *MenuNode
	// synchronize factory and state changes
	mu sync.Mutex
}

func (pd *PooledDialer) Dialer() proxy.ContextDialer {
	if pd.lazy != nil {
		return pd.lazy
	}
	var timeout time.Duration = 1 * time.Hour
	if t := pd.URL.Query().Get("timeout"); t != "" {
		if i, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(i) * time.Second
		}
	}
	pd.lazy = &dialer.Lazy{
		Timeout: timeout,
		New:     pd.factory,
	}
	return pd.lazy
}

func (pd *PooledDialer) AttachMenu(m *MenuGroup) {
	pd.node = m.AddChild(menuet.MenuItem{})
	pd.child = pd.node.AddChild(menuet.MenuItem{})
	pd.update(dialer.Offline)
}

func (pd *PooledDialer) StateChanged(state dialer.ConnectionState, err error) {
	pd.update(state)
	pd.notification(state, err)
}

func (pd *PooledDialer) update(state dialer.ConnectionState) {
	pd.node.Text = pd.icon(state) + " " + pd.Label
	pd.child.Text, pd.child.Clicked = pd.action(state)
}

func (pd *PooledDialer) icon(state dialer.ConnectionState) string {
	switch state {
	case dialer.Offline:
		return "⚪"
	case dialer.Online:
		return "🟢"
	case dialer.Failed:
		return "🔴"
	case dialer.Connecting:
		return "🟡"
	}
	return ""
}

func (pd *PooledDialer) action(state dialer.ConnectionState) (string, func()) {
	switch state {
	case dialer.Offline, dialer.Failed:
		return "Offline", nil
	case dialer.Online, dialer.Connecting:
		return "Disconnect", func() { pd.lazy.Close() }
	}
	return "", nil
}

func (pd *PooledDialer) notification(state dialer.ConnectionState, err error) {
	notif := menuet.Notification{
		Subtitle: pd.Label,
	}
	switch state {
	case dialer.Offline:
		notif.Title = "Proxy disconnected"
		notif.Message = "The connection was terminated."
	case dialer.Connecting:
		notif.Title = "Connecting to proxy"
		notif.Message = "The connection to the proxy is being established."
	case dialer.Online:
		notif.Title = "Proxy connected"
		notif.Message = "The proxy connection has been established."
	case dialer.Failed:
		notif.Title = "Proxy connection failed"
		notif.Message = err.Error()
	default:
		notif.Title = "Unknown connection state"
		notif.Message = "Dialer is in an unknown state."
	}
	if err != nil {
		notif.Message += " " + err.Error()
	}
	menuet.App().Notification(notif)
}

func (pd *PooledDialer) factory() (proxy.ContextDialer, error) {
	pd.mu.Lock()

	pd.StateChanged(dialer.Connecting, nil)
	dd, err := proxy.FromURL(pd.URL, pd.Fwd)
	if err != nil {
		pd.StateChanged(dialer.Failed, err)
		pd.mu.Unlock()
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		err = fmt.Errorf("Dialer does not support DialContext: %s", pd.URL.Scheme)
		pd.StateChanged(dialer.Failed, err)
		pd.mu.Unlock()
		return nil, err
	}
	pd.StateChanged(dialer.Online, nil)

	if w, ok := dd.(interface{ Wait() error }); ok {
		go func() {
			pd.StateChanged(dialer.Offline, w.Wait())
			pd.mu.Unlock()
		}()
	} else {
		pd.mu.Unlock()
	}

	return xd, nil
}
