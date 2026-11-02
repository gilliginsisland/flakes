package dropboxremote

import (
	"context"
	"fmt"
	"io"
	"strings"
)

func SetHead(ctx context.Context, remote, branch string, out, errOut io.Writer) (err error) {
	ctx = debugContext(ctx, errOut)
	git, err := ResolveGit()
	if err != nil {
		return err
	}
	git.errOut = errOut
	debugf(errOut, "setting remote HEAD")
	rawURL, err := git.cmd(ctx, "remote", "get-url", remote).Output()
	if err != nil {
		return commandError(err)
	}
	h, err := connect(ctx, git, strings.TrimSpace(string(rawURL)), out, errOut)
	if err != nil {
		return err
	}
	defer func() { err = redactError(err, h.secret) }()
	ref := "refs/heads/" + branch
	if err := git.cmd(ctx, "check-ref-format", ref).Run(); err != nil {
		return fmt.Errorf("invalid branch %q", branch)
	}
	exists := func() (bool, error) {
		_, err := h.getRefs(ctx, false)
		if err != nil {
			return false, err
		}
		_, ok := h.refs[ref]
		return ok, nil
	}
	if ok, err := exists(); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("remote has no such ref '%s'", ref)
	}
	rev, old, err := h.head(ctx)
	if err != nil {
		return err
	}
	if old == ref {
		return fmt.Errorf("remote HEAD is already '%s'", ref)
	}
	if err := h.writeHead(ctx, ref, rev); err != nil {
		return err
	}
	if ok, err := exists(); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("remote ref '%s' was concurrently deleted: remote HEAD needs to be fixed (try again)", ref)
	}
	fmt.Fprintf(out, "Updated remote HEAD to '%s'.\n", ref)
	return nil
}
