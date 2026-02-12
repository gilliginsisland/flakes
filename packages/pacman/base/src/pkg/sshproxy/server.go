package sshproxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"net"
	"os"

	"github.com/gilliginsisland/pacman/pkg/netutil"
	"golang.org/x/crypto/ssh"
)

type DirectTCPIPPayload struct {
	HostToConnect  string
	PortToConnect  uint32
	OriginatorIP   string
	OriginatorPort uint32
}

// Server is an SSH server that handles proxying through direct-tcpip channels.
type Server struct {
	config  *ssh.ServerConfig
	Dialer  func(ctx context.Context, network, address string) (net.Conn, error)
	HostKey ssh.Signer
}

// loadOrGenerateHostKey loads an existing host key or generates a new one, storing it in the user's home directory.
func LoadOrGenerateHostKey(keyPath string) (ssh.Signer, error) {
	// Check if host key exists
	if _, err := os.Stat(keyPath); err == nil {
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read host key: %v", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %v", err)
		}
		return signer, nil
	}

	// Generate a new host key if it doesn't exist
	private, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from private key: %w", err)
	}

	// Save the private key
	pemBlock, err := ssh.MarshalPrivateKey(private, "")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode the PEM block to a buffer
	var pemBuffer bytes.Buffer
	if err := pem.Encode(&pemBuffer, pemBlock); err != nil {
		return nil, fmt.Errorf("failed to encode PEM block: %v", err)
	}

	if err := os.WriteFile(keyPath, pemBuffer.Bytes(), 0o600); err != nil {
		return nil, fmt.Errorf("failed to save host key: %v", err)
	}

	return signer, nil
}

// Serve starts the SSH server on the given listener.
func (s *Server) Serve(l net.Listener) error {
	// Initialize the SSH server configuration if not already done
	if s.config == nil {
		s.config = &ssh.ServerConfig{
			NoClientAuth: true, // No authentication required
		}
		s.config.AddHostKey(s.HostKey)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("failed to accept connection: %v", err)
		}
		go s.handleConn(conn)
	}
}

// handleConn handles an individual SSH connection.
func (s *Server) handleConn(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		conn.Close()
		return
	}
	defer sshConn.Close()

	// Handle global requests (none are supported)
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChan := range chans {
		if newChan.ChannelType() != "direct-tcpip" {
			newChan.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type %q", newChan.ChannelType()))
			continue
		}

		go s.handleDirectTCPIP(newChan)
	}
}

func (s *Server) handleDirectTCPIP(newChan ssh.NewChannel) {
	// Parse the direct-tcpip request parameters
	var payload DirectTCPIPPayload
	if err := ssh.Unmarshal(newChan.ExtraData(), &payload); err != nil {
		newChan.Reject(ssh.ConnectionFailed, err.Error())
		return
	}

	ch, reqs, err := newChan.Accept()
	if err != nil {
		return
	}
	defer ch.Close()

	// Dial the target using the provided Dialer with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled when function returns

	targetAddr := fmt.Sprintf("%s:%d", payload.HostToConnect, payload.PortToConnect)
	conn, err := s.Dialer(ctx, "tcp", targetAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	// Handle channel requests (e.g., EOF)
	go ssh.DiscardRequests(reqs)

	// Use netutil.Join to pipe data between the SSH channel and the target connection
	// This will close both connections when either one is closed
	netutil.Join(ch, conn)
}
