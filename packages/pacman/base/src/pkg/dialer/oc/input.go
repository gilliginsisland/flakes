package oc

import (
	"context"
	"net/url"
)

// InputRequest describes the credentials requested by the current VPN form.
// Username carries the account already supplied by config or an earlier form.
type InputRequest struct {
	Username     string
	NeedUsername bool
	NeedPassword bool
}

// InputResult holds credentials and an optional action after authentication.
type InputResult struct {
	Username string
	Password string
	// Accepted persists only the reusable password after cookie acquisition.
	Accepted func() error
}

type InputProvider interface {
	Input(ctx context.Context, u *url.URL, request InputRequest) (InputResult, error)
}

// DefaultInputProvider must be registered before dialing.
var DefaultInputProvider InputProvider
