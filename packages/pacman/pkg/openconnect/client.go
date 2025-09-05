package openconnect

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

type Conn struct {
	vpn    *VpnInfo
	cmd    *CMDPipe
	ctx    context.Context
	cancel context.CancelCauseFunc
	once   syncutil.Once
}

func Connect(ctx context.Context, opts Options) (*Conn, error) {
	vpn, err := New(opts)
	if err != nil {
		return nil, err
	}

	conn, err := connect(ctx, vpn)
	if err != nil {
		vpn.Free()
		return nil, err
	}

	return conn, nil
}

func connect(ctx context.Context, vpn *VpnInfo) (*Conn, error) {
	cp, err := vpn.SetupCmdPipe()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancelCause(ctx)
	defer cp.PropagateContext(ctx)()
	defer time.AfterFunc(2*time.Minute, func() {
		cancel(context.DeadlineExceeded)
	}).Stop()

	err = vpn.ObtainCookie()
	if err != nil {
		return nil, err
	}

	err = vpn.MakeCSTPConnection()
	if err != nil {
		return nil, err
	}

	conn := Conn{
		vpn: vpn,
		cmd: cp,
	}
	conn.ctx, conn.cancel = context.WithCancelCause(ctx)
	return &conn, nil
}

func (c *Conn) Run() error {
	c.once.Go(func() {
		c.cancel(c.vpn.MainLoop())
		c.vpn.Free()
	})
	<-c.ctx.Done()
	return context.Cause(c.ctx)
}

func (c *Conn) Wait() error {
	<-c.ctx.Done()
	return context.Cause(c.ctx)
}

func (c *Conn) Close() error {
	c.once.Do(func() {
		// If this executes, Run() was never called, so free resources now
		c.cancel(errors.New("connection closed by user"))
		c.vpn.Free()
	})

	if c.ctx.Err() == nil {
		// Run() was called (or is active)
		// signal termination via command pipe
		c.cmd.Cancel()
	}

	return nil
}

func (c *Conn) TunClient() (*os.File, *IPInfo, error) {
	err := c.TunScript([]string{"builtin"})
	if err != nil {
		return nil, nil, err
	}

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, nil, err
	}
	for _, fd := range fds {
		unix.SetNonblock(fd, true)
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 5<<20)
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, 5<<20)
	}

	defer func() {
		if err != nil {
			unix.Close(fds[0])
			unix.Close(fds[1])
		}
	}()

	oldfd, err := c.vpn.GetTunFd()
	if err == nil {
		unix.Close(oldfd)
	}

	err = c.vpn.SetupTunFd(fds[0])
	if err != nil {
		return nil, nil, err
	}

	ipinfo, err := c.vpn.GetIPInfo()
	if err != nil {
		return nil, nil, err
	}

	return os.NewFile(uintptr(fds[1]), ""), ipinfo, nil
}

func (c *Conn) TunScript(args []string) error {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = strconv.Quote(a)
	}
	script := strings.Join(quoted, " ")

	err := c.vpn.SetupTunScript(script)
	if err != nil {
		return err
	}

	return nil
}
