package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/netutil"
	"sigs.k8s.io/yaml"
)

var ErrProxyNotFound = errors.New("proxy not found")

type Config struct {
	Path    Path
	Listen  netutil.HostPort `json:"listen"`
	Proxies map[string]*URL  `json:"proxies"`
	Rules   []*Rule          `json:"rules"`
}

type Rule struct {
	Hosts   []string `json:"hosts"`
	Proxies []string `json:"proxies"`
}

type URL struct {
	url.URL
}

var _ json.Unmarshaler = (*URL)(nil)

func (u *URL) UnmarshalJSON(data []byte) error {
	{
		var text string
		if err := json.Unmarshal(data, &text); err == nil {
			return u.UnmarshalBinary([]byte(text))
		}
	}

	type Parts struct {
		Username string            `json:"username"`
		Password string            `json:"password"`
		Protocol string            `json:"protocol"`
		Host     string            `json:"host"`
		Path     string            `json:"path"`
		Options  map[string]string `json:"options"`
	}

	var p Parts
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("failed to unmarshal URL: %w", err)
	}

	u.Scheme = p.Protocol
	u.Host = p.Host
	u.Path = path.Clean("/" + p.Path)
	if p.Username != "" || p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	if len(p.Options) > 0 {
		q := url.Values{}
		for k, v := range p.Options {
			q.Add(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return nil
}

// File wraps os.File and implements flag.Value.
type Path string

func (p Path) String() string {
	return string(p)
}

// expandUser expands a leading "~" to the current user's home directory.
func (p Path) ExpandUser() (string, error) {
	s := string(p)
	if !strings.HasPrefix(s, "~") {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand %w", err)
	}
	// replace the leading '~'
	return home + s[1:], nil
}

func ParseConfigFile(path Path) (*Config, error) {
	s, err := path.ExpandUser()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(s)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var rs Config
	err = yaml.Unmarshal(data, &rs)
	if err != nil {
		return nil, err
	}
	rs.Path = path
	return &rs, nil
}
