package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/dialer/ghost"
	"github.com/gilliginsisland/pacman/pkg/matcher"
)

// PacHandler generates a PAC file for browser proxy configuration.
type PacHandler struct {
	Rules ghost.Ruleset
}

func (h *PacHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")

	w.Write([]byte("function FindProxyForURL(url, host) {\n"))

	for _, rule := range h.Rules {
		cond := make([]string, len(rule.Hosts))
		for i, m := range rule.Hosts {
			cond[i] = toJSCondition(m)
		}

		prox := make([]string, len(rule.Proxies))
		for i, u := range rule.Proxies {
			var usePacMan bool

			switch u.Scheme {
			case "http", "https", "socks5":
				usePacMan = h.Rules.MatchHost(u.Hostname()) != nil
			default:
				usePacMan = true
			}

			if usePacMan {
				prox[i] = "PROXY " + r.Host
			} else {
				prox[i] = toPACDirective(&u.URL)
			}
		}

		if len(cond) > 0 && len(prox) > 0 {
			cond := strings.Join(cond, " || ")
			prox := strings.Join(prox, "; ")
			w.Write(
				fmt.Appendf(nil, "\tif (%s) return \"%s\";\n", cond, prox),
			)
		}
	}

	w.Write([]byte("\treturn \"DIRECT\";\n"))
	w.Write([]byte("}\n"))
}

func toJSCondition(hm ghost.HostMatcher) string {
	switch m := hm.StringMatcher.(type) {
	case matcher.Literal:
		// Exact match
		return fmt.Sprintf("host == \"%s\"", string(m))

	case *matcher.CIDR:
		// Use `isInNet(host, network, mask)` PAC function
		network := m.Network.IP.String()
		mask := net.IP(m.Network.Mask).String()
		return fmt.Sprintf("isInNet(host, \"%s\", \"%s\")", network, mask)

	case *regexp.Regexp:
		// Use JavaScript's `new RegExp` constructor
		pattern, _ := json.Marshal(m.String())
		return fmt.Sprintf("new RegExp(%s).test(host)", pattern)
	}

	return "false"
}

func toPACDirective(u *url.URL) string {
	scheme := strings.ToUpper(u.Scheme)
	if scheme == "HTTP" {
		scheme = "PROXY"
	}

	d := fmt.Sprintf("%s %s", scheme, u.Host)

	// safari requires SOCKS not SOCKS5
	if scheme == "SOCKS5" {
		d += fmt.Sprintf("; SOCKS %s", u.Host)
	}

	return d
}
