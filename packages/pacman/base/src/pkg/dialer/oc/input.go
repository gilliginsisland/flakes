package oc

import (
	"context"
	"net/url"
)

type InputType int

const (
	InputPassword InputType = iota
)

// InputProvider supplies the complete authentication value for the requested
// input type and URL being connected.
type InputProvider interface {
	Input(ctx context.Context, u *url.URL, input InputType) (string, error)
}

// DefaultInputProvider must be registered before dialing. A nil provider cannot
// fulfill authentication forms that require input.
var DefaultInputProvider InputProvider
