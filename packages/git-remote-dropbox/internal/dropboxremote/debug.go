package dropboxremote

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"golang.org/x/oauth2"
)

func debugf(out io.Writer, format string, args ...any) {
	if out != nil && os.Getenv("GIT_REMOTE_DROPBOX_DEBUG") == "1" {
		fmt.Fprintf(out, "git-remote-dropbox: "+format+"\n", args...)
	}
}

func debugResult(out io.Writer, operation string, err error, secrets ...string) {
	if err != nil {
		debugf(out, "%s failed: type=%T error=%v", operation, err, redactError(err, secrets...))
	} else {
		debugf(out, "%s completed", operation)
	}
}

type debugTransport struct {
	base http.RoundTripper
	out  io.Writer
}

func (t debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	debugf(t.out, "HTTP %s %s%s started", req.Method, req.URL.Host, req.URL.Path)
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		// Transport errors can contain URLs. Report the type without query values.
		debugf(t.out, "HTTP %s %s%s failed: type=%T elapsed=%s", req.Method, req.URL.Host, req.URL.Path, err, time.Since(start))
	} else {
		debugf(t.out, "HTTP %s %s%s status=%d request_id=%q elapsed=%s", req.Method, req.URL.Host, req.URL.Path, resp.StatusCode, resp.Header.Get("X-Dropbox-Request-Id"), time.Since(start))
	}
	return resp, err
}

func debugContext(ctx context.Context, out io.Writer) context.Context {
	if os.Getenv("GIT_REMOTE_DROPBOX_DEBUG") != "1" {
		return ctx
	}
	client := &http.Client{Transport: debugTransport{base: http.DefaultTransport, out: out}}
	return context.WithValue(ctx, oauth2.HTTPClient, client)
}

func debugDropbox(ctx context.Context, cfg dropbox.Config) dropbox.Config {
	if os.Getenv("GIT_REMOTE_DROPBOX_DEBUG") == "1" {
		source := cfg.TokenSource
		if source == nil {
			source = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
		}
		cfg.Client = oauth2.NewClient(ctx, source)
	}
	return cfg
}
