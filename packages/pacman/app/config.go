package app

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

var ErrProxyNotFound = errors.New("proxy not found")

type Config struct {
	Path    Path
	Proxies map[string]*URL `json:"proxies"`
	Rules   []*Rule         `json:"rules"`
}

type Rule struct {
	Hosts   []string `json:"hosts"`
	Proxies []string `json:"proxies"`
}

type URL struct {
	url.URL
}

var _ encoding.TextUnmarshaler = (*URL)(nil)

func (u *URL) UnmarshalText(text []byte) error {
	return u.UnmarshalBinary(text)
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
