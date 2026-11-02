package dropboxremote

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type session struct {
	ctx    context.Context
	remote *helper
	input  *bufio.Scanner
}

type pushRefspec struct {
	source, destination string
}

// readLine keeps protocol reads cancellable even when stdin is a blocking pipe.
func (s *session) readLine() (string, error) {
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	type result struct {
		line string
		err  error
	}
	ready := make(chan result, 1)
	go func() {
		if s.input.Scan() {
			ready <- result{line: s.input.Text()}
		} else if err := s.input.Err(); err != nil {
			ready <- result{err: err}
		} else {
			ready <- result{err: io.EOF}
		}
	}()
	select {
	case <-s.ctx.Done():
		return "", s.ctx.Err()
	case next := <-ready:
		return next.line, next.err
	}
}

func (s *session) run() error {
	s.input.Buffer(make([]byte, 4096), 1024*1024)
	for {
		line, err := s.readLine()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if line == "" {
			return nil
		}
		if err := s.dispatch(line); err != nil {
			return err
		}
	}
}

func (s *session) dispatch(line string) error {
	command, _, _ := strings.Cut(line, " ")
	switch command {
	case "capabilities", "option", "list", "push", "fetch":
		debugf(s.remote.errOut, "received %s command", command)
	}
	switch {
	case line == "capabilities":
		s.capabilities()
		return nil
	case strings.HasPrefix(line, "option "):
		return s.option(line)
	case line == "list" || line == "list for-push":
		return s.list(line == "list for-push")
	case strings.HasPrefix(line, "push "):
		return s.pushBatch(line)
	case strings.HasPrefix(line, "fetch "):
		return s.fetchBatch(line)
	default:
		return fmt.Errorf("unsupported operation: %s", line)
	}
}

func (s *session) capabilities() {
	fmt.Fprint(s.remote.out, "option\npush\nfetch\n\n")
	debugf(s.remote.errOut, "sent capabilities")
}

func (s *session) option(line string) error {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		fmt.Fprintln(s.remote.out, "unsupported")
		return nil
	}
	switch fields[1] {
	case "verbosity":
		verbosity, err := strconv.Atoi(fields[2])
		if err != nil || verbosity < 0 {
			fmt.Fprintln(s.remote.out, "error invalid value")
			return nil
		}
		debugf(s.remote.errOut, "setting verbosity=%d", verbosity)
		s.remote.verbosity = verbosity
	case "progress":
		switch fields[2] {
		case "true":
			s.remote.showProgress = true
		case "false":
			s.remote.showProgress = false
		default:
			fmt.Fprintln(s.remote.out, "error invalid value")
			return nil
		}
		debugf(s.remote.errOut, "setting progress=%t", s.remote.showProgress)
	default:
		fmt.Fprintln(s.remote.out, "unsupported")
		return nil
	}
	fmt.Fprintln(s.remote.out, "ok")
	return nil
}

func (s *session) list(forPush bool) error {
	debugf(s.remote.errOut, "listing remote references (for push=%t)", forPush)
	names, err := s.remote.getRefs(s.ctx, forPush)
	if err != nil {
		return err
	}
	debugf(s.remote.errOut, "listed %d remote references", len(names))
	for _, name := range names {
		fmt.Fprintf(s.remote.out, "%s %s\n", s.remote.refs[name].sha, name)
	}
	if !forPush {
		_, head, err := s.remote.head(s.ctx)
		if err != nil {
			return err
		}
		if head != "" {
			fmt.Fprintf(s.remote.out, "@%s HEAD\n", head)
		}
	}
	fmt.Fprintln(s.remote.out)
	return nil
}

func (s *session) pushBatch(line string) error {
	head := ""
	for {
		spec, err := s.push(line)
		if err != nil {
			return err
		}
		if s.remote.firstPush && spec.source != "" {
			localHead, _ := s.remote.git.cmd(s.ctx, "symbolic-ref", "HEAD").Output()
			if head == "" || strings.TrimPrefix(spec.source, "+") == strings.TrimSpace(string(localHead)) {
				head = spec.destination
			}
		}
		next, err := s.readLine()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if next == "" {
			break
		}
		line = next
	}
	debugf(s.remote.errOut, "push batch finished")
	s.finishPush(head)
	return nil
}

func (s *session) push(line string) (pushRefspec, error) {
	refspec, ok := strings.CutPrefix(line, "push ")
	if !ok {
		return pushRefspec{}, fmt.Errorf("invalid push command")
	}
	source, destination, ok := strings.Cut(refspec, ":")
	if !ok {
		return pushRefspec{}, fmt.Errorf("invalid push refspec")
	}
	debugf(s.remote.errOut, "pushing reference")
	if err := s.remote.push(s.ctx, source, destination); err != nil {
		if s.ctx.Err() != nil {
			return pushRefspec{}, s.ctx.Err()
		}
		err = redactError(err, s.remote.secret)
		fmt.Fprintf(s.remote.errOut, "error: push failed: %v\n", err)
		fmt.Fprintf(s.remote.out, "error %s %s\n", destination, strings.ReplaceAll(err.Error(), "\n", " "))
		return pushRefspec{}, nil
	}
	debugf(s.remote.errOut, "push completed")
	fmt.Fprintf(s.remote.out, "ok %s\n", destination)
	return pushRefspec{source: source, destination: destination}, nil
}

func (s *session) finishPush(head string) {
	if s.remote.firstPush {
		s.remote.firstPush = false
		if head != "" {
			debugf(s.remote.errOut, "setting first push HEAD=%q", head)
			if err := s.remote.writeHead(s.ctx, head, ""); err != nil {
				fmt.Fprintf(s.remote.errOut, "info: failed to set default branch: %v\n", redactError(err, s.remote.secret))
			}
		}
	}
	fmt.Fprintln(s.remote.out)
}

func (s *session) fetchBatch(line string) error {
	for {
		if err := s.fetch(line); err != nil {
			return err
		}
		next, err := s.readLine()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if next == "" {
			break
		}
		line = next
	}
	fmt.Fprintln(s.remote.out)
	return nil
}

func (s *session) fetch(line string) error {
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != "fetch" {
		return fmt.Errorf("invalid fetch command")
	}
	debugf(s.remote.errOut, "fetching objects")
	err := s.remote.fetch(s.ctx, fields[1])
	debugf(s.remote.errOut, "fetch ended (failed=%t)", err != nil)
	return err
}
