package app

import (
	"encoding"
	"errors"
	"io"
	"net/url"
	"os"

	"sigs.k8s.io/yaml"
)

var ErrProxyNotFound = errors.New("proxy not found")

type RuleSet struct {
	Path    string
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

func (p *URL) UnmarshalText(text []byte) error {
	return p.UnmarshalBinary(text)
}

func LoadRuleSetFile(path string) (*RuleSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var rs RuleSet
	err = yaml.Unmarshal(data, &rs)
	if err != nil {
		return nil, err
	}
	rs.Path = string(path)
	return &rs, nil
}
