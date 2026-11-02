package dropboxremote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"golang.org/x/sync/errgroup"
)

type remoteRef struct {
	rev, sha string
}

type helper struct {
	git          Git
	secret       string
	store        storage
	root         string
	refs         map[string]remoteRef
	pushed       map[string]string
	firstPush    bool
	verbosity    int
	showProgress bool
	out, errOut  io.Writer
}

func (h *helper) progress(kind string, done, total int, final bool) {
	if !h.showProgress {
		return
	}
	if final {
		fmt.Fprintf(h.errOut, "\r%s objects: 100%% (%d/%d), done.\n", kind, done, total)
	} else {
		fmt.Fprintf(h.errOut, "\r%s objects: %d%% (%d/%d)", kind, done*100/max(total, 1), done, total)
	}
}

func (h *helper) objectPath(sha string) (string, error) {
	if len(sha) != 40 || strings.Trim(sha, "0123456789abcdef") != "" {
		return "", fmt.Errorf("invalid SHA-1 object ID %q", sha)
	}
	return path.Join(h.root, "objects", sha[:2], sha[2:]), nil
}

func (h *helper) refPath(ref string) (string, error) {
	if !strings.HasPrefix(ref, "refs/") || path.Clean(ref) != ref || strings.ContainsAny(ref, "\r\n\\") {
		return "", fmt.Errorf("invalid reference %q", ref)
	}
	return path.Join(h.root, ref), nil
}

func (h *helper) getRefs(ctx context.Context, forPush bool) ([]string, error) {
	entries, err := h.store.list(ctx, path.Join(h.root, "refs"))
	if errors.Is(err, errNotFound) {
		clear(h.refs)
		if forPush {
			h.firstPush = true
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	values := make([]remoteRef, len(entries))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(20)
	for i, entry := range entries {
		group.Go(func() error {
			name := strings.TrimPrefix(entry.PathLower, h.root+"/")
			if _, err := h.refPath(name); err != nil {
				return err
			}
			rev, data, err := h.store.get(ctx, entry.PathLower)
			if err != nil {
				return err
			}
			sha := strings.TrimSpace(string(data))
			if _, err := h.objectPath(sha); err != nil {
				return err
			}
			names[i], values[i] = name, remoteRef{rev, sha}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	clear(h.refs)
	for i, name := range names {
		h.refs[name] = values[i]
	}
	sort.Strings(names)
	return names, nil
}

func (h *helper) head(ctx context.Context) (string, string, error) {
	rev, data, err := h.store.get(ctx, path.Join(h.root, "HEAD"))
	if errors.Is(err, errNotFound) {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	ref, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "ref: ")
	if !ok {
		return "", "", fmt.Errorf("invalid remote HEAD")
	}
	if _, err := h.refPath(ref); err != nil {
		return "", "", err
	}
	return rev, ref, nil
}

func (h *helper) writeHead(ctx context.Context, ref, rev string) error {
	if _, err := h.refPath(ref); err != nil {
		return err
	}
	mode := writeMode(files.WriteModeAdd, "")
	if rev != "" {
		mode = writeMode(files.WriteModeUpdate, rev)
	}
	return h.store.put(ctx, path.Join(h.root, "HEAD"), []byte("ref: "+ref+"\n"), mode)
}

func (h *helper) push(ctx context.Context, src, dst string) (err error) {
	defer func() { debugResult(h.errOut, "push reference", err, h.secret) }()
	debugf(h.errOut, "push source=%q destination=%q", src, dst)
	dstPath, err := h.refPath(dst)
	if err != nil {
		return err
	}
	if err := h.git.cmd(ctx, "check-ref-format", dst).Run(); err != nil {
		return fmt.Errorf("invalid reference %q", dst)
	}
	if src == "" {
		_, head, err := h.head(ctx)
		if err != nil {
			return err
		}
		if head == dst {
			return fmt.Errorf("refusing to delete the current branch: %s", head)
		}
		if err := h.store.delete(ctx, dstPath); err != nil {
			return err
		}
		delete(h.refs, dst)
		delete(h.pushed, dst)
		return nil
	}
	force := strings.HasPrefix(src, "+")
	src = strings.TrimPrefix(src, "+")
	shaData, err := h.git.cmd(ctx, "rev-parse", "--verify", src).Output()
	if err != nil {
		return commandError(err)
	}
	sha := strings.TrimSpace(string(shaData))
	if _, err := h.objectPath(sha); err != nil {
		return err
	}
	args := []string{"rev-list", "--objects", sha}
	for _, ref := range h.refs {
		if err := h.git.cmd(ctx, "cat-file", "-e", ref.sha).Run(); err == nil {
			args = append(args, "^"+ref.sha)
		}
	}
	for _, pushed := range h.pushed {
		args = append(args, "^"+pushed)
	}
	objects, err := h.git.cmd(ctx, args...).Output()
	if err != nil {
		return commandError(err)
	}
	var ids []string
	for _, line := range strings.Split(string(objects), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			ids = append(ids, fields[0])
		}
	}
	if err := h.uploadObjects(ctx, ids); err != nil {
		return err
	}
	mode := writeMode(files.WriteModeAdd, "")
	if force {
		mode = writeMode(files.WriteModeOverwrite, "")
	} else if old, ok := h.refs[dst]; ok {
		if err := h.git.cmd(ctx, "cat-file", "-e", old.sha).Run(); err != nil {
			return fmt.Errorf("fetch first")
		}
		if err := h.git.cmd(ctx, "merge-base", "--is-ancestor", old.sha, sha).Run(); err != nil {
			return fmt.Errorf("non-fast forward")
		}
		mode = writeMode(files.WriteModeUpdate, old.rev)
	}
	debugf(h.errOut, "publishing reference destination=%q mode=%s", dst, mode.Tag)
	if err := h.store.put(ctx, dstPath, []byte(sha+"\n"), mode); err != nil {
		return err
	}
	h.pushed[dst] = sha
	return nil
}

func (h *helper) fetch(ctx context.Context, sha string) (err error) {
	debugf(h.errOut, "fetch object=%s started", sha)
	defer func() { debugResult(h.errOut, "fetch objects", err, h.secret) }()
	if _, err := h.objectPath(sha); err != nil {
		return err
	}
	seen := make(map[string]bool)
	queue := []string{sha}
	done := 0
	for len(queue) > 0 {
		debugf(h.errOut, "fetch batch objects=%d done=%d", len(queue), done)
		batch := queue
		queue = nil
		children := make([][]string, len(batch))
		group, fetchCtx := errgroup.WithContext(ctx)
		group.SetLimit(20)
		for i, id := range batch {
			if seen[id] {
				continue
			}
			seen[id] = true
			group.Go(func() error {
				if err := h.git.cmd(fetchCtx, "cat-file", "-e", id).Run(); err == nil {
					if id == "4b825dc642cb6eb9a060e54bf8d69288fbee4904" {
						if _, err := h.git.write(fetchCtx, "tree", []byte{}); err != nil {
							return err
						}
					}
					if err := h.git.cmd(fetchCtx, "rev-list", "--objects", id).Run(); err == nil {
						return nil
					}
				} else {
					objectPath, err := h.objectPath(id)
					if err != nil {
						return err
					}
					_, data, err := h.store.get(fetchCtx, objectPath)
					if err != nil {
						return err
					}
					if err := h.git.decode(fetchCtx, data, id); err != nil {
						return err
					}
				}
				refs, err := h.git.referenced(fetchCtx, id)
				children[i] = refs
				return err
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
		done += len(batch)
		for _, refs := range children {
			for _, ref := range refs {
				if !seen[ref] {
					queue = append(queue, ref)
				}
			}
		}
		h.progress("Receiving", done, done+len(queue), len(queue) == 0)
	}
	return nil
}
