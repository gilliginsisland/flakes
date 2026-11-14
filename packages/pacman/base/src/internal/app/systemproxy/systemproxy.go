// Package systemproxy synchronizes PACman's system proxy switch and settings.
package systemproxy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
	"github.com/gilliginsisland/pacman/pkg/menuet/security"
	sc "github.com/gilliginsisland/pacman/pkg/menuet/systemconfiguration"
	"github.com/gilliginsisland/pacman/pkg/notify"
)

var errChangeCanceled = errors.New("proxy change canceled")

type proxyType string

const (
	proxyPAC       proxyType = "Automatic Proxy Configuration"
	proxyHTTP      proxyType = "HTTP"
	proxyHTTPS     proxyType = "HTTPS"
	proxySOCKS     proxyType = "SOCKS"
	proxyFTP       proxyType = "FTP"
	proxyGopher    proxyType = "Gopher"
	proxyRTSP      proxyType = "RTSP"
	proxyDiscovery proxyType = "Automatic Proxy Discovery"
)

// Keep a stable order when displaying the settings and applying changes.
var proxyKeys = [...]proxyType{
	proxyPAC,
	proxyHTTP,
	proxyHTTPS,
	proxySOCKS,
	proxyFTP,
	proxyGopher,
	proxyRTSP,
	proxyDiscovery,
}

// Each proxy pairs its enable key with its address key. Automatic discovery has
// no address setting. Dictionary keys are borrowed native framework constants.
var proxies = map[proxyType]struct {
	enableKey  cf.StringRef
	addressKey cf.StringRef
}{
	proxyPAC:       {sc.PropNetProxiesProxyAutoConfigEnable, sc.PropNetProxiesProxyAutoConfigURLString},
	proxyHTTP:      {sc.PropNetProxiesHTTPEnable, sc.PropNetProxiesHTTPProxy},
	proxyHTTPS:     {sc.PropNetProxiesHTTPSEnable, sc.PropNetProxiesHTTPSProxy},
	proxySOCKS:     {sc.PropNetProxiesSOCKSEnable, sc.PropNetProxiesSOCKSProxy},
	proxyFTP:       {sc.PropNetProxiesFTPEnable, sc.PropNetProxiesFTPProxy},
	proxyGopher:    {sc.PropNetProxiesGopherEnable, sc.PropNetProxiesGopherProxy},
	proxyRTSP:      {sc.PropNetProxiesRTSPEnable, sc.PropNetProxiesRTSPProxy},
	proxyDiscovery: {sc.PropNetProxiesProxyAutoDiscoveryEnable, 0},
}

// Switch is the control used to display and change the system proxy state.
type Switch interface {
	State() appkit.ControlStateValue
	SetState(appkit.ControlStateValue)
	Enabled() bool
	SetEnabled(bool)
	SetSpinning(bool)
	SetCallback(*core.Callback)
	Menu() *appkit.Menu
}

// Proxy synchronizes an existing switch with the effective PAC
// settings. Its observer remains scheduled for the lifetime of the application.
type Proxy struct {
	mu    sync.Mutex
	ctrl  Switch
	cb    *core.Callback
	url   string
	store *sc.DynamicStore
	auth  *security.Authorization
	prefs *sc.Preferences
}

// New observes the system proxy and synchronizes the existing switch.
func New(ctrl Switch, addr net.Addr) (*Proxy, error) {
	proxy := Proxy{ctrl: ctrl}
	if proxy.url = pacURL(addr); proxy.url == "" {
		return nil, errors.New("cannot create a PAC URL from the listener address")
	}
	var err error
	if proxy.auth, err = security.NewAuthorization(); err != nil {
		return nil, err
	}
	if proxy.prefs, err = sc.NewPreferencesWithAuthorization("PACman Proxy Settings", proxy.auth); err != nil {
		return nil, err
	}
	if proxy.store, err = sc.NewDynamicStore("PACman Proxy Observer", proxy.sync); err != nil {
		return nil, err
	}
	key := sc.CreateProxiesKey()
	defer cf.Release(cf.TypeRef(key))
	if err = proxy.store.SetNotificationKeys(key); err != nil {
		return nil, err
	}
	proxy.cb = core.NewCallback(func(sender core.Sender) {
		core.Release(sender.Pointer())
		go proxy.toggle()
	})
	proxy.ctrl.SetCallback(proxy.cb)
	if err := proxy.store.Schedule(); err != nil {
		return nil, err
	}
	proxy.sync()
	return &proxy, nil
}

// sync reconciles the switch under the mutex and notifies on an On-to-Off transition.
func (proxy *Proxy) sync() {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()
	proxy.ctrl.SetEnabled(false)
	defer proxy.ctrl.SetEnabled(true)
	proxy.ctrl.SetSpinning(true)
	defer proxy.ctrl.SetSpinning(false)
	previous := proxy.ctrl.State()
	config, err := proxy.store.CopyProxies()
	if err != nil {
		return
	}
	matched := config.Get(proxies[proxyPAC].addressKey).String() == proxy.url &&
		config.Get(sc.PropNetProxiesProxyAutoConfigJavaScript).String() == ""
	if matched {
		for _, key := range proxyKeys {
			if matched = config.Get(proxies[key].enableKey).Bool() == (key == proxyPAC); !matched {
				break
			}
		}
	}
	state := appkit.ControlStateValueOff
	if matched {
		state = appkit.ControlStateValueOn
	}
	proxy.ctrl.SetState(state)
	if previous == appkit.ControlStateValueOn && !matched {
		proxyNotification("System Proxy Changed Outside PACman", "PACman's system proxy control was disabled because the proxy settings no longer match PACman.")
	}
}

func (proxy *Proxy) toggle() {
	proxy.mu.Lock()
	proxy.ctrl.SetEnabled(false)
	proxy.ctrl.SetSpinning(true)
	enabled := proxy.ctrl.State() == appkit.ControlStateValueOn
	var err error
	defer func() {
		if err == nil {
			proxy.ctrl.SetSpinning(false)
			proxy.ctrl.SetEnabled(true)
		}
		proxy.mu.Unlock()
		if err != nil {
			proxy.sync()
			if err != errChangeCanceled {
				proxyNotification("System Proxy Not Updated", err.Error())
			}
		}
	}()
	if menu := proxy.ctrl.Menu(); menu != nil {
		menu.CancelTracking()
	}
	if enabled {
		var config *cf.Dictionary
		config, err = proxy.store.CopyProxies()
		if err != nil {
			return
		}
		if !confirm(config, proxy.url) {
			err = errChangeCanceled
			return
		}
	}
	err = proxy.SetEnabled(enabled)
}

// SetEnabled changes only system preferences. Observed settings drive the switch.
func (proxy *Proxy) SetEnabled(enabled bool) error {
	if err := proxy.auth.CopyRight("system.preferences.network",
		security.AuthorizationFlagExtendRights|security.AuthorizationFlagInteractionAllowed); err != nil {
		return fmt.Errorf("permission to change the system proxy was not granted: %w", err)
	}
	pac := proxies[proxyPAC]
	// Invalidate the cache before locking; read preferences only after locking.
	proxy.prefs.Synchronize()
	if err := proxy.prefs.Lock(false); err != nil {
		return err
	}
	defer proxy.prefs.Unlock()
	set, err := proxy.prefs.CopyCurrentSet()
	if err != nil {
		return err
	}
	count := 0
	for _, id := range set.ServiceOrder() {
		service := proxy.prefs.CopyService(id)
		if service == nil || !service.Enabled() {
			continue
		}
		protocol := service.CopyProtocol(sc.NetworkProtocolTypeProxies)
		if protocol == nil {
			continue
		}
		saved := protocol.Configuration()
		if saved == nil {
			continue
		}
		count++
		config := saved.MutableCopy()
		if enabled {
			config.SetString(pac.addressKey, proxy.url)
			config.Remove(sc.PropNetProxiesProxyAutoConfigJavaScript)
			for _, candidate := range proxyKeys {
				config.SetBool(proxies[candidate].enableKey, candidate == proxyPAC)
			}
		} else {
			if config.Get(pac.addressKey).String() != proxy.url {
				continue
			}
			config.SetBool(pac.enableKey, false)
			config.Remove(pac.addressKey)
		}
		if err = protocol.SetConfiguration(&config.Dictionary); err != nil {
			return err
		}
		if enabled {
			if err = protocol.SetEnabled(true); err != nil {
				return err
			}
		}
	}
	if count == 0 {
		return errors.New("no enabled network services with proxy support were found in the current network location")
	}
	if err = proxy.prefs.CommitChanges(); err != nil {
		return err
	}
	if err = proxy.prefs.ApplyChanges(); err != nil {
		return fmt.Errorf("proxy settings were saved, but macOS could not apply them; check System Settings before retrying: %w", err)
	}
	return nil
}

// confirmationText explains conflicts in the effective proxy settings.
func confirmationText(config *cf.Dictionary, url string) string {
	var parts []string
	if config.Get(sc.PropNetProxiesProxyAutoConfigJavaScript).String() != "" {
		parts = append(parts, "An existing inline proxy configuration script will be replaced by PACman's PAC URL.")
	}
	var active []string
	for _, key := range proxyKeys {
		proxy := proxies[key]
		var address string
		if proxy.addressKey != 0 {
			address = config.Get(proxy.addressKey).String()
		}
		if key == proxyPAC {
			if address != "" && address != url {
				parts = append(parts, "Setting PACman will replace the "+string(key)+" URL:\n\n"+address)
			}
			continue
		}
		if config.Get(proxy.enableKey).Bool() {
			label := string(key)
			if address != "" {
				label += " (" + address + ")"
			}
			active = append(active, label)
		}
	}
	if len(active) > 0 {
		parts = append(parts, fmt.Sprintf("Active proxy settings: %s. Setting PACman as the system proxy will disable these settings. Their saved addresses and ports will be kept.", strings.Join(active, ", ")))
	}
	return strings.Join(parts, "\n\n")
}

func confirm(config *cf.Dictionary, url string) bool {
	text := confirmationText(config, url)
	if text == "" {
		return true
	}
	alert := appkit.NewAlert().
		SetMessageText("Set PACman as the System Proxy?").
		SetInformativeText(text).
		AddButtonWithTitle("Set").
		AddButtonWithTitle("Cancel")
	return alert.RunModal() == appkit.AlertFirstButtonReturn
}

func proxyNotification(subtitle, body string) {
	notify.Notify(notify.Notification{Identifier: "system-proxy", Subtitle: subtitle, Body: body})
}

func pacURL(addr net.Addr) string {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || tcp.Port <= 0 {
		return ""
	}
	ip := tcp.IP
	if len(ip) == 0 || ip.IsUnspecified() {
		if ip.To4() != nil || len(ip) == 0 {
			ip = net.IPv4(127, 0, 0, 1)
		} else {
			ip = net.IPv6loopback
		}
	}
	u := url.URL{Scheme: "http", Host: net.JoinHostPort(ip.String(), strconv.Itoa(tcp.Port)), Path: "/proxy.pac"}
	return u.String()
}
