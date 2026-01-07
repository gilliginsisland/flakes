package openconnect

/*
#cgo pkg-config: openconnect

#cgo noescape go_vpninfo_new
#cgo noescape go_mainloop
#cgo noescape openconnect_vpninfo_free
#cgo noescape openconnect_set_useragent
#cgo noescape openconnect_parse_url
#cgo noescape openconnect_get_hostname
#cgo noescape openconnect_set_hostname
#cgo noescape openconnect_get_protocol
#cgo noescape openconnect_set_protocol
#cgo noescape openconnect_setup_tun_script
#cgo noescape openconnect_setup_tun_fd
#cgo noescape openconnect_set_loglevel
#cgo noescape openconnect_setup_csd
#cgo noescape openconnect_setup_dtls
#cgo noescape openconnect_disable_dtls
#cgo noescape openconnect_set_dpd
#cgo noescape openconnect_obtain_cookie
#cgo noescape openconnect_setup_cmd_pipe
#cgo noescape openconnect_make_cstp_connection

#cgo nocallback go_mainloop

#include <openconnect.h>
#include <stdlib.h>

extern struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata);
extern int go_mainloop(struct openconnect_info *vpninfo);
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var handles = syncutil.Map[uintptr, *VpnInfo]{}

type Protocol string

const (
	ProtocolAnyConnect    Protocol = "anyconnect"
	ProtocolGlobalProtect Protocol = "gp"
)

var UserAgents = map[Protocol]string{
	ProtocolAnyConnect:    "AnyConnect Darwin_i386 5.1.8.122",
	ProtocolGlobalProtect: "Global Protect",
}

type Callbacks struct {
	ValidatePeerCert   func(cert string) bool
	ProcessAuthForm    func(form *AuthForm) FormResult
	Progress           func(level LogLevel, message string)
	ExternalBrowser    func(uri string) error
	ReconnectedHandler func()
}

type Options struct {
	UserAgent           string
	Protocol            Protocol
	Server              string
	CSD                 string
	LogLevel            LogLevel
	ForceDPD            int
	AllowInsecureCrypto bool
	Callbacks
}

// VpnInfo represents a VPN session in Go.
type VpnInfo struct {
	vpninfo *C.struct_openconnect_info
	done    chan struct{}
	err     syncutil.AtomicValue[error]
	Callbacks
}

// NewVpnInfo initializes a new VPN session with callbacks.
func New(opts Options) (*VpnInfo, error) {
	if opts.UserAgent == "" {
		opts.UserAgent, _ = UserAgents[opts.Protocol]
	}

	cUserAgent := C.CString(opts.UserAgent)
	defer C.free(unsafe.Pointer(cUserAgent))

	vpninfo := C.go_vpninfo_new(cUserAgent, nil)
	if vpninfo == nil {
		return nil, errors.New("failed to create VPN session")
	}

	v := VpnInfo{
		vpninfo: vpninfo,
		done:    make(chan struct{}),
	}

	handles.Store(uintptr(unsafe.Pointer(vpninfo)), &v)

	err := v.ParseOpts(opts)
	if err != nil {
		v.Free()
		return nil, err
	}

	return &v, nil
}

// Free cleans up the VPN session.
func (v *VpnInfo) Free() {
	if v.vpninfo == nil {
		return
	}
	C.openconnect_vpninfo_free(v.vpninfo)
	handles.Delete(uintptr(unsafe.Pointer(v.vpninfo)))
	v.vpninfo = nil
	select {
	case <-v.done:
	default:
		close(v.done)
	}
}

func (v *VpnInfo) ParseOpts(opts Options) error {
	v.Callbacks = opts.Callbacks

	v.SetLogLevel(opts.LogLevel)

	if opts.UserAgent != "" {
		if err := v.SetUserAgent(opts.UserAgent); err != nil {
			return err
		}
	}

	if opts.Protocol != "" {
		if err := v.SetProtocol(opts.Protocol); err != nil {
			return err
		}
	}

	if opts.Server != "" {
		if err := v.ParseURL(opts.Server); err != nil {
			return err
		}
	}

	if opts.CSD != "" {
		if err := v.SetupCSD(os.Getuid(), true, opts.CSD); err != nil {
			return err
		}
	}

	if opts.ForceDPD > 0 {
		v.SetDPD(opts.ForceDPD)
	}

	if opts.AllowInsecureCrypto {
		if err := v.SetAllowInsecureCrypto(opts.AllowInsecureCrypto); err != nil {
			return err
		}
	}

	return nil
}

// SetUserAgent sets the VPN user agent string.
func (v *VpnInfo) SetUserAgent(userAgent string) error {
	cStr := C.CString(userAgent)
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("set user agent", C.openconnect_set_useragent(v.vpninfo, cStr))
}

func (v *VpnInfo) ParseURL(url string) error {
	cStr := C.CString(url)
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("parse URL", C.openconnect_parse_url(v.vpninfo, cStr))
}

// Hostname returns the VPN server hostname.
func (v *VpnInfo) Hostname() string {
	return C.GoString(C.openconnect_get_hostname(v.vpninfo))
}

// SetHostname sets the VPN server hostname.
func (v *VpnInfo) SetHostname(hostname string) error {
	cStr := C.CString(hostname)
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("set hostname", C.openconnect_set_hostname(v.vpninfo, cStr))
}

// Protocol returns the VPN protocol.
func (v *VpnInfo) Protocol() Protocol {
	return Protocol(C.GoString(C.openconnect_get_protocol(v.vpninfo)))
}

// SetProtocol sets the VPN protocol.
func (v *VpnInfo) SetProtocol(protocol Protocol) error {
	cStr := C.CString(string(protocol))
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("set protocol", C.openconnect_set_protocol(v.vpninfo, cStr))
}

func (v *VpnInfo) SetupTunScript(script string) error {
	cStr := C.CString(script)
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("setup tun script", C.openconnect_setup_tun_script(v.vpninfo, cStr))
}

func (v *VpnInfo) SetupTunFd(fd int) error {
	return ocErrno("setup tun fd", C.openconnect_setup_tun_fd(v.vpninfo, C.int(fd)))
}

func (v *VpnInfo) SetLogLevel(level LogLevel) {
	C.openconnect_set_loglevel(v.vpninfo, C.int(level))
}

func (v *VpnInfo) SetupCSD(uid int, silent bool, wrapper string) error {
	cWrapper := C.CString(wrapper)
	defer C.free(unsafe.Pointer(cWrapper))

	var silentC C.int
	if silent {
		silentC = 1
	}

	return ocErrno("setup CSD", C.openconnect_setup_csd(v.vpninfo, C.uid_t(uid), silentC, cWrapper))
}

func (v *VpnInfo) SetAllowInsecureCrypto(allowed bool) error {
	var allowedC C.uint
	if allowed {
		allowedC = 1
	}
	return ocErrno("set allow-insecure-crypto", C.openconnect_set_allow_insecure_crypto(v.vpninfo, allowedC))
}

func (v *VpnInfo) SetupDTLS(attemptPeriod int) error {
	return ocErrno("setup DTLS", C.openconnect_setup_dtls(v.vpninfo, C.int(attemptPeriod)))
}

func (v *VpnInfo) DisableDTLS() error {
	return ocErrno("disable DTLS", C.openconnect_disable_dtls(v.vpninfo))
}

func (v *VpnInfo) SetDPD(min_seconds int) {
	C.openconnect_set_dpd(v.vpninfo, C.int(min_seconds))
}

func (v *VpnInfo) ObtainCookie() error {
	return ocErrno("obtain cookie", C.openconnect_obtain_cookie(v.vpninfo))
}

func (v *VpnInfo) SetupCmdPipe() (*CMDPipe, error) {
	fd := C.openconnect_setup_cmd_pipe(v.vpninfo)
	if fd < 0 {
		return nil, ocErrno("setup cmd pipe", fd)
	}
	f := os.NewFile(uintptr(fd), "")
	return &CMDPipe{w: f}, nil
}

func (v *VpnInfo) MakeCSTPConnection() error {
	return ocErrno("make CSTP connection", C.openconnect_make_cstp_connection(v.vpninfo))
}

func (v *VpnInfo) MainLoop() error {
	return ocErrno("main loop", C.go_mainloop(v.vpninfo))
}

func (v *VpnInfo) GetTunFd() (int, error) {
	return -1, fmt.Errorf("get tun fd: %w", syscall.ENOTSUP)
}

// Err returns the error from mainloop if it has completed
// or nil if it hasn’t run or isn’t done
func (v *VpnInfo) Err() error {
	return v.err.Load()
}

// Done returns a channel that is closed when the mainloop completes
func (v *VpnInfo) Done() <-chan struct{} {
	return v.done
}

func ocErrno(context string, rc C.int) error {
	if rc == 0 {
		return nil
	}
	return fmt.Errorf("%s: %w", context, syscall.Errno(-rc))
}
