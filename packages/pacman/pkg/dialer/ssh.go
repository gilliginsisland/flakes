package dialer

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/gilliginsisland/pacman/internal/netutil"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("ssh", NewSSHFromURL)
}

type SSH struct {
	address string
	config  *ssh.ClientConfig
	fwd     proxy.Dialer
}

func NewSSHFromURL(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "22"
	}

	config := &ssh.ClientConfig{
		User:            u.User.Username(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	query := u.Query()

	if password, found := u.User.Password(); found {
		config.Auth = append(config.Auth, ssh.Password(password))
	}

	if filename := query.Get("identity"); filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read IdentityFile: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IdentityFile: %w", err)
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	return &SSH{
		fwd:     forward,
		address: net.JoinHostPort(host, port),
		config:  config,
	}, nil
}

func (d *SSH) Dial(network, address string) (net.Conn, error) {
	return netutil.DialContext(nil, d.fwd, network, address)
}

func (d *SSH) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := netutil.DialContext(ctx, d.fwd, "tcp", d.address)
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, d.address, d.config)
	if err != nil {
		return nil, err
	}

	client := ssh.NewClient(clientConn, chans, reqs)
	return client.Dial(network, address)
}
