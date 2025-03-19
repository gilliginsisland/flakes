package dialer

import (
	"net/url"

	"golang.org/x/net/proxy"
)

type Factory struct {
	cache   map[string]proxy.Dialer
	forward proxy.Dialer
}

func NewFactory(fwd proxy.Dialer) *Factory {
	return &Factory{
		cache:   make(map[string]proxy.Dialer),
		forward: fwd,
	}
}

func (f *Factory) Get(p string) (proxy.Dialer, error) {
	if d, ok := f.cache[p]; ok {
		return d, nil
	}

	u, err := url.Parse(p)
	if err != nil {
		return nil, err
	}

	d, err := proxy.FromURL(u, f.forward)
	if err != nil {
		return nil, err
	}

	f.cache[p] = d
	return d, nil
}
