package dialer

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

func init() {
	RegisterContextDialerType("ssh", SSH)
}

func SSH(ctx context.Context, u *url.URL, fwd proxy.Dialer) (proxy.Dialer, error) {
	config := ssh.ClientConfig{
		User:            u.User.Username(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "22"
	}
	addr := net.JoinHostPort(host, port)

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

	ctx, cancel := context.WithCancel(ctx)

	conn, err := dialContext(ctx, fwd, "tcp", addr)
	if err != nil {
		cancel()
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, &config)
	if err != nil {
		conn.Close()
		cancel()
		return nil, err
	}
	go func() {
		clientConn.Wait()
		cancel()
	}()

	return ssh.NewClient(clientConn, chans, reqs), nil
}
