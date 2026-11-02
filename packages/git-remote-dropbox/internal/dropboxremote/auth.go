package dropboxremote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/auth"
	dropboxoauth "github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/oauth"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/retry"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/users"
	"golang.org/x/oauth2"
)

const oauthClientID = "42zjexze6mfpf7x"

func oauthConfig(ctx context.Context, token *oauth2.Token) dropbox.Config {
	return debugDropbox(ctx, dropbox.Config{
		TokenSource: dropboxoauth.TokenSource(ctx, oauthClientID, token),
		RetryPolicy: &retry.Policy{MaxRetries: 3, RetryRateLimitsIndefinitely: true},
	})
}

func (g Git) authenticate(ctx context.Context, c *credential, out io.Writer) (cfg dropbox.Config, err error) {
	defer func() { err = redactError(err, c.Password) }()
	debugf(out, "looking up stored credential")
	fillErr := g.fillCredential(ctx, c)
	debugResult(out, "credential lookup", fillErr, c.Password)
	debugf(out, "credential lookup finished (failed=%t, token present=%t)", fillErr != nil, c.Password != "")
	if fillErr == nil && c.Password != "" {
		debugf(out, "validating stored credential with Dropbox")
		cfg := oauthConfig(ctx, &oauth2.Token{RefreshToken: c.Password})
		_, err := users.NewContext(cfg).GetCurrentAccountContext(ctx)
		if err == nil {
			debugf(out, "stored credential accepted; approving credential")
			return cfg, g.approveCredential(ctx, *c)
		}
		debugResult(out, "stored credential validation", err, c.Password)
		if !invalidCredential(err) {
			return dropbox.Config{}, err
		}
		debugf(out, "stored credential invalid; rejecting credential")
		if err := g.rejectCredential(ctx, *c); err != nil {
			return dropbox.Config{}, err
		}
	}
	if ctx.Err() != nil {
		return dropbox.Config{}, ctx.Err()
	}
	debugf(out, "starting OAuth authorization")
	token, err := g.authorize(ctx, out)
	if err != nil {
		return dropbox.Config{}, err
	}
	debugf(out, "validating new credential with Dropbox")
	cfg = oauthConfig(ctx, token)
	if _, err := users.NewContext(cfg).GetCurrentAccountContext(ctx); err != nil {
		return dropbox.Config{}, redactError(err, token.AccessToken, token.RefreshToken)
	}
	debugf(out, "new credential accepted; approving credential")
	c.Password = token.RefreshToken
	if err := g.approveCredential(ctx, *c); err != nil {
		return dropbox.Config{}, err
	}
	return cfg, nil
}

func (g Git) authorize(ctx context.Context, out io.Writer) (*oauth2.Token, error) {
	flow, err := dropboxoauth.NewPKCEFlow(oauthClientID)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(out, "Authorize Dropbox access at:\n%s\nThen enter the authorization code at Git's password prompt.\n", flow.AuthCodeURL())
	debugf(out, "waiting for authorization code through Git")
	code, err := g.authorizationCode(ctx)
	if err != nil {
		return nil, err
	}
	debugf(out, "exchanging authorization code with Dropbox")
	token, err := flow.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("Dropbox authorization failed: %w", redactError(err, code, flow.Verifier()))
	}
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("Dropbox did not return a refresh token")
	}
	return token, nil
}

func (g Git) authorizationCode(ctx context.Context) (string, error) {
	code := credential{Protocol: credentialProtocol, Host: credentialHost, Username: "authorization-code"}
	input, err := code.encode()
	if err != nil {
		return "", err
	}
	// Disable storage helpers for this one-time code. Git still handles askpass
	// and terminal input, leaving remote-helper stdin untouched.
	cmd := g.cmd(ctx, "-c", "credential.helper=", "credential", "fill")
	cmd.Stdin = strings.NewReader(input)
	data, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not read Dropbox authorization code: %w", commandError(err, code.Password))
	}
	if err := code.decode(data); err != nil {
		return "", err
	}
	if code.Password == "" {
		return "", fmt.Errorf("Git did not provide a Dropbox authorization code")
	}
	return strings.TrimSpace(code.Password), nil
}

func invalidCredential(err error) bool {
	var apiErr auth.AuthAPIError
	if errors.As(err, &apiErr) && apiErr.AuthError != nil {
		return apiErr.AuthError.Tag == auth.AuthErrorInvalidAccessToken || apiErr.AuthError.Tag == auth.AuthErrorExpiredAccessToken
	}
	var oauthErr *oauth2.RetrieveError
	return errors.As(err, &oauthErr) && oauthErr.ErrorCode == "invalid_grant"
}
