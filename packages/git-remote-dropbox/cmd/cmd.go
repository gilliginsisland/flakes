package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gilliginsisland/flakes/git-remote-dropbox/internal/dropboxremote"
	"io"
	"os"
	"os/signal"
	"syscall"
)

var Version = "dev"

func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	flags := flag.NewFlagSet("git-remote-dropbox", flag.ContinueOnError)
	flags.SetOutput(errOut)
	version := flags.Bool("version", false, "Print version")
	branch := flags.String("set-head", "", "Set the default branch on the named remote")
	flags.Usage = func() {
		fmt.Fprintln(errOut, "Usage: git-remote-dropbox [flags] <remote> <url>\n       git remote-dropbox --set-head <branch> <remote>")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *version {
		if flags.NArg() != 0 || *branch != "" {
			return fmt.Errorf("--version does not accept other arguments")
		}
		fmt.Fprintf(out, "git-remote-dropbox %s\n", Version)
		return nil
	}
	if *branch != "" {
		if flags.NArg() != 1 {
			return fmt.Errorf("usage: git-remote-dropbox --set-head <branch> <remote>")
		}
		return dropboxremote.SetHead(ctx, flags.Arg(0), *branch, out, errOut)
	}
	return dropboxremote.RunHelper(ctx, flags.Args(), in, out, errOut)
}

func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	if err := Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
