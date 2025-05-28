package openconnect

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type Conn struct {
	vpn    *VpnInfo
	cmd    *CMDPipe
	done   chan struct{}
	cancel context.CancelFunc
	once   sync.Once
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

	go func() {
		conn.Wait()
		vpn.Free()
	}()

	return conn, nil
}

func connect(ctx context.Context, vpn *VpnInfo) (*Conn, error) {
	cp, err := vpn.SetupCmdPipe()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cp.PropagateContext(ctx)()

	conn := Conn{
		vpn:    vpn,
		cmd:    cp,
		done:   make(chan struct{}),
		cancel: cancel,
	}

	defer time.AfterFunc(2*time.Minute, cancel).Stop()

	err = vpn.ObtainCookie()
	if err != nil {
		return nil, err
	}

	err = vpn.MakeCSTPConnection()
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func (c *Conn) Run() error {
	var err error
	c.once.Do(func() {
		c.vpn.MainLoop()
		close(c.done)
	})
	return err
}

func (c *Conn) Wait() error {
	<-c.done
	return nil
}

func (c *Conn) Close() error {
	c.cancel()
	c.once.Do(func() {
		close(c.done)
	})
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

	fd, err := c.vpn.SwapTunFd(fds[0])
	if err != nil {
		return nil, nil, err
	}
	unix.Close(fd)

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
