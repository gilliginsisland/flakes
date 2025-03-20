package dialer

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gilliginsisland/pacman/internal/netutil"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

func init() {
	proxy.RegisterDialerType("ssh", NewSSHFromURL)
}

type SSHDialerConfig struct {
	ssh.ClientConfig
	Address  string
	Dialer   proxy.Dialer
	IdleTime time.Duration
}

type SSH struct {
	config *SSHDialerConfig
	client *ssh.Client
	mu     sync.RWMutex
}

func NewSSH(config *SSHDialerConfig) *SSH {
	s := SSH{config: config}
	return &s
}

func NewSSHFromURL(u *url.URL, fwd proxy.Dialer) (proxy.Dialer, error) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "22"
	}

	config := SSHDialerConfig{
		Address: net.JoinHostPort(host, port),
		ClientConfig: ssh.ClientConfig{
			User:            u.User.Username(),
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		Dialer: fwd,
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

	return NewSSH(&config), nil
}

func (s *SSH) Dial(network, address string) (net.Conn, error) {
	return s.DialContext(nil, network, address)
}

func (s *SSH) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	client, err := s.getClient()
	if err != nil {
		return nil, err
	}
	return client.DialContext(ctx, network, address)
}

func (s *SSH) getClient() (*ssh.Client, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client != nil {
		return client, nil
	}

	// acquire full lock to create a new client
	s.mu.Lock()
	defer s.mu.Unlock()

	// check again in case another goroutine already created the client
	if s.client != nil {
		return s.client, nil
	}

	client, err := s.connect()
	if err != nil {
		return nil, err
	}

	s.client = client

	// Monitor client connection in a separate goroutine
	go s.monitor(client)

	return client, nil
}

// connect establishes a new SSH connection.
func (s *SSH) connect() (*ssh.Client, error) {
	conn, err := netutil.DialContext(context.Background(), s.config.Dialer, "tcp", s.config.Address)
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, s.config.Address, &s.config.ClientConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return ssh.NewClient(clientConn, chans, reqs), nil
}

func (s *SSH) monitor(client *ssh.Client) {
	// wait for the client to shutdown
	_ = client.Wait()

	// now acquire a write lock to clear the client.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == client {
		s.client = nil
	}
}
