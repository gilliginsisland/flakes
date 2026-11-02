package dropboxremote

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	credentialProtocol = "https"
	credentialHost     = "dropbox.com"
)

type credential struct {
	Protocol, Host, Username, Password string
}

func (c credential) encode() (string, error) {
	for _, value := range []string{
		c.Protocol, c.Host, c.Username, c.Password,
	} {
		if strings.ContainsAny(value, "\r\n\x00") {
			return "", fmt.Errorf("invalid credential field")
		}
	}
	var input strings.Builder
	fmt.Fprintf(&input, "protocol=%s\nhost=%s\n", c.Protocol, c.Host)
	if c.Username != "" {
		fmt.Fprintf(&input, "username=%s\n", c.Username)
	}
	if c.Password != "" {
		fmt.Fprintf(&input, "password=%s\n", c.Password)
	}
	input.WriteByte('\n')
	return input.String(), nil
}

func (g Git) fillCredential(ctx context.Context, c *credential) (err error) {
	debugf(g.errOut, "fillCredential started")
	defer func() { debugResult(g.errOut, "fillCredential", err, c.Password) }()
	input, err := c.encode()
	if err != nil {
		return err
	}
	cmd := g.cmd(ctx, "-c", "credential.interactive=false", "-c", "core.askPass=", "credential", "fill")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=")
	cmd.Stdin = strings.NewReader(input)
	// Credential stdout stays private; report stderr with known secrets redacted.
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git credential fill failed: %w", commandError(err, c.Password))
	}
	return c.decode(data)
}

func (c *credential) decode(data []byte) error {
	result := *c
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		name, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		switch name {
		case "protocol":
			result.Protocol = value
		case "host":
			result.Host = value
		case "username":
			result.Username = value
		case "password":
			result.Password = value
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("invalid credential helper response")
	}
	*c = result
	return nil
}

func (g Git) approveCredential(ctx context.Context, c credential) (err error) {
	debugf(g.errOut, "approveCredential started")
	defer func() { debugResult(g.errOut, "approveCredential", err, c.Password) }()
	input, err := c.encode()
	if err != nil {
		return err
	}
	cmd := g.cmd(ctx, "credential", "approve")
	cmd.Stdin = strings.NewReader(input)
	if _, err = cmd.Output(); err != nil {
		return fmt.Errorf("git credential approve failed: %w", commandError(err, c.Password))
	}
	return nil
}

func (g Git) rejectCredential(ctx context.Context, c credential) (err error) {
	debugf(g.errOut, "rejectCredential started")
	defer func() { debugResult(g.errOut, "rejectCredential", err, c.Password) }()
	input, err := c.encode()
	if err != nil {
		return err
	}
	cmd := g.cmd(ctx, "credential", "reject")
	cmd.Stdin = strings.NewReader(input)
	if _, err = cmd.Output(); err != nil {
		return fmt.Errorf("git credential reject failed: %w", commandError(err, c.Password))
	}
	return nil
}
