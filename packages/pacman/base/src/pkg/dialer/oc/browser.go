package oc

import "context"

// BrowserProvider runs browser authentication until the browser closes or ctx
// ends, returning the session error or ctx.Err().
type BrowserProvider interface {
	Browser(ctx context.Context, uri string) error
}

// DefaultBrowserProvider must be registered before browser authentication is requested.
var DefaultBrowserProvider BrowserProvider
