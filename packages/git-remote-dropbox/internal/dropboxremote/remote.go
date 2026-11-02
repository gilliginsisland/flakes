package dropboxremote

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/retry"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/users"
)

func connect(ctx context.Context, git Git, rawURL string, out, errOut io.Writer) (_ *helper, err error) {
	debugf(errOut, "parsing remote URL")
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Dropbox URL")
	}
	if u.Scheme != "dropbox" {
		return nil, fmt.Errorf("URL must start with the \"dropbox://\" scheme")
	}
	if u.Host != "" {
		return nil, fmt.Errorf("URL with no username or token must start with \"dropbox:///\"")
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasSuffix(u.Path, "/") {
		return nil, fmt.Errorf("URL path must be absolute and must not have trailing slash")
	}
	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	if username != "" && password != "" {
		return nil, fmt.Errorf("URL must not specify both username and token")
	}
	if username == "" {
		username = "default"
	}
	c := credential{Protocol: credentialProtocol, Host: credentialHost, Username: username, Password: password}
	defer func() { err = redactError(err, password, c.Password) }()
	cfg := debugDropbox(ctx, dropbox.Config{Token: password, RetryPolicy: &retry.Policy{MaxRetries: 3, RetryRateLimitsIndefinitely: true}})
	if password == "" {
		cfg, err = git.authenticate(ctx, &c, errOut)
	} else {
		debugf(errOut, "validating inline access token")
		_, err = users.NewContext(cfg).GetCurrentAccountContext(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("Dropbox connection failed: %w", err)
	}

	debugf(errOut, "Dropbox connection ready")
	return &helper{
		git:       git,
		secret:    c.Password,
		store:     storage{api: files.NewContext(cfg), errOut: errOut, secret: c.Password},
		root:      strings.ToLower(u.Path),
		refs:      make(map[string]remoteRef),
		pushed:    make(map[string]string),
		verbosity: 1,
		out:       out,
		errOut:    errOut,
	}, nil
}

func RunHelper(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	ctx = debugContext(ctx, errOut)
	executable, _ := os.Executable()
	debugf(errOut, "starting helper executable=%q", executable)
	if len(args) != 2 {
		return fmt.Errorf("usage: git-remote-dropbox <remote> <url>")
	}
	git, err := ResolveGit()
	if err != nil {
		return err
	}
	git.errOut = errOut
	debugf(errOut, "Git executable=%q", git.Executable)
	h, err := connect(ctx, git, args[1], out, errOut)
	if err != nil {
		return err
	}
	session := session{ctx: ctx, remote: h, input: bufio.NewScanner(in)}
	debugf(errOut, "starting protocol session")
	err = session.run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	err = redactError(err, h.secret)
	debugResult(errOut, "protocol session", err, h.secret)
	return err
}
