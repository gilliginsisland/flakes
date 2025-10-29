package openconnect

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
	"golang.org/x/sys/unix"
)

type Conn struct {
	vpn  *VpnInfo
	cmd  *CMDPipe
	once syncutil.Once
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

	defer cp.PropagateContext(ctx)()

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
	return &conn, nil
}

func (c *Conn) Run() error {
	var err error
	ran := c.once.Do(func() {
		err = c.vpn.MainLoop()
	})
	if !ran {
		return errors.New("cannot reuse completed connection")
	}
	return err
}

func (c *Conn) Wait() error {
	<-c.vpn.Done()
	return c.vpn.Err()
}

func (c *Conn) Close() error {
	if c.once.Do(c.vpn.Free) {
		return nil
	}
	select {
	case <-c.vpn.Done():
		return nil
	default:
		return c.cmd.Cancel()
	}
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
