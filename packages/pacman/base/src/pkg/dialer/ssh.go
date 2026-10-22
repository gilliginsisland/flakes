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

func SSH(ctx context.Context, u *url.URL, fwd proxy.Dialer) (Dialer, error) {
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

	conn, err := DialContext(ctx, fwd, "tcp", addr)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, &config)
	// The factory cancels ctx after returning. Detach its handshake-only close
	// hook before handing ownership of conn to the long-lived SSH client.
	if !stop() {
		return nil, ctx.Err()
	}
	if err != nil {
		conn.Close()
		return nil, err
	}

	return ssh.NewClient(clientConn, chans, reqs), nil
}
