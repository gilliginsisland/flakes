package dropboxremote

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Git struct {
	Executable string
	errOut     io.Writer
}

// ResolveGit uses the core executable directory supplied by the invoking Git.
func ResolveGit() (Git, error) {
	dir := os.Getenv("GIT_EXEC_PATH")
	if dir == "" {
		return Git{}, fmt.Errorf("GIT_EXEC_PATH is not set; invoke this helper through Git")
	}
	if !filepath.IsAbs(dir) {
		return Git{}, fmt.Errorf("GIT_EXEC_PATH must be an absolute directory path")
	}
	name := "git"
	if runtime.GOOS == "windows" {
		name = "git.exe"
	}
	return Git{Executable: filepath.Join(dir, name)}, nil
}

func (g Git) cmd(ctx context.Context, args ...string) *exec.Cmd {
	// Only log known command names: arbitrary arguments may contain remote tokens.
	for _, arg := range args {
		switch arg {
		case "credential", "cat-file", "hash-object", "ls-tree", "check-ref-format", "rev-parse", "rev-list", "merge-base", "symbolic-ref", "remote":
			debugf(g.errOut, "Git command=%s arguments=%d", arg, len(args))
			return exec.CommandContext(ctx, g.Executable, args...)
		}
	}
	debugf(g.errOut, "Git command arguments=%d", len(args))
	return exec.CommandContext(ctx, g.Executable, args...)
}

func (g Git) decode(ctx context.Context, data []byte, expected string) (err error) {
	debugf(g.errOut, "Git decode started")
	defer func() { debugResult(g.errOut, "Git decode", err) }()
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	header, contents, ok := bytes.Cut(raw, []byte{0})
	if !ok {
		return fmt.Errorf("invalid loose object header")
	}
	kind, size, ok := strings.Cut(string(header), " ")
	n, err := strconv.Atoi(size)
	if !ok || err != nil || n != len(contents) {
		return fmt.Errorf("invalid loose object size")
	}
	sha, err := g.write(ctx, kind, contents)
	if err != nil {
		return err
	}
	if sha != expected {
		return fmt.Errorf("hash mismatch %s != %s", sha, expected)
	}
	return nil
}

func (g Git) write(ctx context.Context, kind string, data []byte) (_ string, err error) {
	debugf(g.errOut, "Git write started")
	defer func() { debugResult(g.errOut, "Git write", err) }()
	cmd := g.cmd(ctx, "hash-object", "-w", "--stdin", "-t", kind)
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), commandError(err)
}

func (g Git) referenced(ctx context.Context, sha string) (_ []string, err error) {
	debugf(g.errOut, "Git referenced started")
	defer func() { debugResult(g.errOut, "Git referenced", err) }()
	kindData, err := g.cmd(ctx, "cat-file", "-t", sha).Output()
	if err != nil {
		return nil, commandError(err)
	}
	kind := strings.TrimSpace(string(kindData))
	var refs []string
	switch kind {
	case "blob":
		return nil, nil
	case "tree":
		// NUL-delimited output preserves arbitrary filenames, including newlines.
		data, err := g.cmd(ctx, "ls-tree", "-z", sha).Output()
		if err != nil {
			return nil, commandError(err)
		}
		for entry := range bytes.SplitSeq(data, []byte{0}) {
			metadata, _, ok := bytes.Cut(entry, []byte{'\t'})
			if !ok {
				continue
			}
			fields := strings.Fields(string(metadata))
			if len(fields) != 3 {
				return nil, fmt.Errorf("invalid tree entry")
			}
			if fields[0] != "160000" {
				refs = append(refs, fields[2])
			}
		}
	case "commit", "tag":
		data, err := g.cmd(ctx, "cat-file", kind, sha).Output()
		if err != nil {
			return nil, commandError(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if line == "" {
				break
			}
			key, value, _ := strings.Cut(line, " ")
			if key == "tree" || key == "parent" || (kind == "tag" && key == "object") {
				refs = append(refs, value)
			}
		}
	default:
		return nil, fmt.Errorf("unexpected object type %q", kind)
	}
	return refs, nil
}

func (g Git) streamObjects(ctx context.Context, ids []string, queue chan<- uploadObject) (err error) {
	debugf(g.errOut, "streaming %d Git objects", len(ids))
	defer func() { debugResult(g.errOut, "Git object stream", err) }()
	cmd := g.cmd(ctx, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(strings.Join(ids, "\n") + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			cmd.Process.Kill()
		}
		waitErr := cmd.Wait()
		if err == nil && waitErr != nil {
			err = fmt.Errorf("git cat-file failed: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
		}
	}()
	reader := bufio.NewReader(output)
	for i, sha := range ids {
		data, err := readLooseObject(reader, sha)
		if err != nil {
			return err
		}
		select {
		case queue <- uploadObject{index: i, sha: sha, data: data}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func readLooseObject(reader *bufio.Reader, sha string) ([]byte, error) {
	header, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(header)
	if len(fields) != 3 || fields[0] != sha {
		return nil, fmt.Errorf("invalid git cat-file response for %s", sha)
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size < 0 {
		return nil, fmt.Errorf("invalid git object size")
	}
	var data bytes.Buffer
	compressed := zlib.NewWriter(&data)
	if _, err := fmt.Fprintf(compressed, "%s %s%c", fields[1], fields[2], 0); err != nil {
		return nil, err
	}
	if _, err := io.CopyN(compressed, reader, size); err != nil {
		compressed.Close()
		return nil, err
	}
	if err := compressed.Close(); err != nil {
		return nil, err
	}
	if newline, err := reader.ReadByte(); err != nil || newline != '\n' {
		return nil, fmt.Errorf("invalid git object terminator")
	}
	return data.Bytes(), nil
}
