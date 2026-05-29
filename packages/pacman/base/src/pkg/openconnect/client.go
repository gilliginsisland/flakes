package openconnect

import (
	"context"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type Conn struct {
	vpn      *VpnInfo
	cmd      *CMDPipe
	ctx      context.Context
	cancel   context.CancelCauseFunc
	mainLoop bool
}

const (
	CiscoVersionString     = "5.1.8.122"
	CiscoUserAgent         = "AnyConnect Darwin_i386 " + CiscoVersionString
	GlobalProtectUserAgent = "Global Protect"
)

func Connect(ctx context.Context, opts Options) (*Conn, error) {
	switch opts.Protocol {
	case ProtocolAnyConnect:
		if opts.UserAgent == "" {
			opts.UserAgent = CiscoUserAgent
		}
		if opts.VersionString == "" {
			opts.VersionString = CiscoVersionString
		}
	case ProtocolGlobalProtect:
		if opts.UserAgent == "" {
			opts.UserAgent = GlobalProtectUserAgent
		}
	}

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

	defer context.AfterFunc(ctx, func() {
		cp.Cancel()
	})()

	if err = vpn.ObtainCookie(); err != nil {
		return nil, err
	}

	if err = vpn.MakeCSTPConnection(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	conn := Conn{
		vpn:    vpn,
		cmd:    cp,
		ctx:    ctx,
		cancel: cancel,
	}
	context.AfterFunc(ctx, vpn.Free)
	return &conn, nil
}

func (c *Conn) Run() error {
	done := make(chan struct{})
	go func() {
		select {
		case err := <-c.vpn.errCh:
			c.cancel(err)
		case <-done:
		}
	}()
	err := c.vpn.MainLoop()
	if err != nil {
		close(done)
		c.cancel(err)
	} else {
		c.mainLoop = true
	}
	return err
}

func (c *Conn) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Conn) Wait() error {
	<-c.Done()
	return context.Cause(c.ctx)
}

func (c *Conn) Close() error {
	if c.mainLoop {
		return c.cmd.Cancel()
	}
	c.cancel(nil)
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
