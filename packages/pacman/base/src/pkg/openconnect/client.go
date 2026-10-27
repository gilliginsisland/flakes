package openconnect

/*
#include <shm_bridge.h>
*/
import "C"

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/shmring"
)

type connState uint8

const (
	connIdle connState = iota
	connRunning
	connCloseRequested
)

// Conn must be configured before Run or from its synchronous setup-tunnel
// callback. State transitions determine whether the worker or Close cancels ctx,
// which schedules native cleanup.
type Conn struct {
	vpn    *VpnInfo
	cmd    *CMDPipe
	ctx    context.Context
	cancel context.CancelCauseFunc
	mu     sync.Mutex
	state  connState
}

const (
	CiscoVersionString     = "5.1.8.122"
	CiscoUserAgent         = "AnyConnect Darwin_i386 " + CiscoVersionString
	GlobalProtectUserAgent = "Global Protect"
)

func Connect(ctx context.Context, opts Options) (*Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

	stop := context.AfterFunc(ctx, func() {
		cp.Cancel()
	})
	defer stop()

	defer func() {
		if err != nil {
			cp.Close()
		}
	}()

	if err = vpn.ObtainCookie(); err != nil {
		return nil, err
	}
	if err = vpn.MakeCSTPConnection(); err != nil {
		return nil, err
	}
	// CSTP negotiates the keys for AnyConnect DTLS or GlobalProtect ESP.
	// Start UDP before Run so the main loop can complete its establishment.
	if dtlsErr := vpn.SetupDTLS(60); dtlsErr != nil {
		if err = vpn.DisableDTLS(); err != nil {
			return nil, err
		}
		if vpn.Progress != nil {
			vpn.Progress(LogLevelInfo, "Set up UDP failed; using SSL instead: "+dtlsErr.Error())
		}
	}
	if !stop() {
		err = ctx.Err()
		return nil, err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	context.AfterFunc(ctx, func() {
		cp.Close()
		vpn.Free()
	})
	return &Conn{
		vpn:    vpn,
		cmd:    cp,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Run starts the native main loop.
func (c *Conn) Run() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := context.Cause(c.ctx); err != nil {
		return err
	}
	switch c.state {
	case connRunning:
		return errors.New("connection already in progress")
	case connCloseRequested:
		return os.ErrClosed
	}
	c.state = connRunning
	go func() {
		// Once started, the native loop must return before cleanup is scheduled.
		c.cancel(c.vpn.MainLoop())
	}()
	return nil
}

// Done closes when native work has stopped; resource cleanup follows asynchronously.
func (c *Conn) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Wait returns the shutdown cause after native work has stopped.
func (c *Conn) Wait() error {
	<-c.Done()
	return context.Cause(c.ctx)
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == connCloseRequested || c.ctx.Err() != nil {
		return os.ErrClosed
	}
	state := c.state
	c.state = connCloseRequested
	if state == connRunning {
		return c.cmd.Cancel()
	}
	c.cancel(nil)
	return nil
}

// TunClient configures the shared packet rings before packet processing starts,
// either before Run or from the native setup-tunnel callback.
func (c *Conn) TunClient() (io.ReadWriteCloser, *IPInfo, error) {
	ipinfo, err := c.vpn.GetIPInfo()
	if err != nil {
		return nil, nil, err
	}

	if ipinfo.MTU == 0 || ipinfo.MTU > 65535 {
		return nil, nil, &OpError{Op: "setup packet bridge", Err: syscall.EINVAL}
	}
	endpoints, err := shmring.NewBridgePair(256, max(16384, ipinfo.MTU))
	if err != nil {
		return nil, nil, err
	}
	client, native := shmring.NewReadWriter(shmring.NewBridge(endpoints[0])), (*C.struct_shm_bridge)(unsafe.Pointer(endpoints[1]))
	go func() {
		// OpenConnect borrows its endpoint until native packet processing stops.
		select {
		case <-client.Done():
			c.Close()
			c.Wait()
		case <-c.Done():
			client.Close()
		}
		C.shm_bridge_close(native)
	}()
	if err := c.vpn.SetupPacketBridge(native); err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, ipinfo, nil
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
