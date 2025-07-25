package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/dialer/ghost"
)

func jsString(s string) string {
	// Escape special characters for JavaScript strings
	js, _ := json.Marshal(s)
	return string(js)
}

// PacHandler generates a PAC file for browser proxy configuration.
type PacHandler struct {
	Rules ghost.RuleSet
}

func (h *PacHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")

	var checks []string

	fmt.Fprintln(w, "function FindProxyForURL(url, host) {")
	fmt.Fprintln(w, "\tswitch (host) {")
	fmt.Fprintln(w, "\tdefault:")
	fmt.Fprintln(w, "\t\tbreak;")
	for _, rule := range h.Rules {
		for _, h := range rule.Hosts {
			if strings.HasPrefix(h, "*.") {
				suffix := strings.TrimPrefix(h, "*")
				checks = append(checks, fmt.Sprintf("host.substring(host.length - %d) === %s", len(suffix), jsString(suffix)))
			} else if _, ipnet, err := net.ParseCIDR(h); err == nil {
				checks = append(checks, fmt.Sprintf("isInNet(host, \"%s\", \"%s\")", ipnet.IP.String(), net.IP(ipnet.Mask).String()))
			} else {
				fmt.Fprintf(w, "\tcase %s:\n", jsString(h))
			}
		}
	}
	fmt.Fprintf(w, "\t\treturn \"PROXY %s\";\n", r.Host)
	fmt.Fprintln(w, "\t}")

	fmt.Fprintf(w, "\tif (%s) {\n", strings.Join(checks, " || "))
	fmt.Fprintf(w, "\t\treturn \"PROXY %s\";\n", r.Host)
	fmt.Fprintln(w, "\t}")

	fmt.Fprintln(w, "\treturn \"DIRECT\";")
	fmt.Fprintln(w, "}")
}
