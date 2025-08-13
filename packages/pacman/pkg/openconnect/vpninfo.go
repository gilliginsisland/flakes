package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
#include <stdlib.h>

extern struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata);
*/
import "C"

import (
	"errors"
	"os"
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
	ProtocolAnyConnect:    "AnyConnect Darwin_i386 4.10.01075",
	ProtocolGlobalProtect: "Global Protect",
}

type Callbacks struct {
	ValidatePeerCert func(cert string) bool
	ProcessAuthForm  func(form AuthForm) FormResult
	Progress         func(level LogLevel, message string)
	ExternalBrowser  func(uri string) error
}

type Options struct {
	UserAgent string
	Protocol  Protocol
	Server    string
	CSD       string
	LogLevel  LogLevel
	ForceDPD  int
	Callbacks
}

// VpnInfo represents a VPN session in Go.
type VpnInfo struct {
	vpninfo *C.struct_openconnect_info
	Callbacks
}

// NewVpnInfo initializes a new VPN session with callbacks.
func New(opts Options) (*VpnInfo, error) {
	if opts.UserAgent == "" {
		opts.UserAgent, _ = UserAgents[opts.Protocol]
	}

	cUserAgent := C.CString(opts.UserAgent)
	defer C.free(unsafe.Pointer(cUserAgent))

	v := VpnInfo{}

	v.vpninfo = C.go_vpninfo_new(cUserAgent, nil)
	if v.vpninfo == nil {
		return nil, errors.New("failed to create VPN session")
	}

	handles.Store(uintptr(unsafe.Pointer(v.vpninfo)), &v)

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
		if err := v.SetProtocol(string(opts.Protocol)); err != nil {
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

	return nil
}

// SetUserAgent sets the VPN user agent string.
func (v *VpnInfo) SetUserAgent(userAgent string) error {
	cStr := C.CString(userAgent)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_set_useragent(v.vpninfo, cStr) != 0 {
		return errors.New("failed to set user agent")
	}
	return nil
}

func (v *VpnInfo) ParseURL(url string) error {
	cStr := C.CString(url)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_parse_url(v.vpninfo, cStr) != 0 {
		return errors.New("failed to parse URL")
	}
	return nil
}

// Hostname returns the VPN server hostname.
func (v *VpnInfo) Hostname() string {
	return C.GoString(C.openconnect_get_hostname(v.vpninfo))
}

// SetHostname sets the VPN server hostname.
func (v *VpnInfo) SetHostname(hostname string) error {
	cStr := C.CString(hostname)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_set_hostname(v.vpninfo, cStr) != 0 {
		return errors.New("failed to set hostname")
	}
	return nil
}

// Protocol returns the VPN protocol.
func (v *VpnInfo) Protocol() string {
	return C.GoString(C.openconnect_get_protocol(v.vpninfo))
}

// SetProtocol sets the VPN protocol.
func (v *VpnInfo) SetProtocol(protocol string) error {
	cStr := C.CString(protocol)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_set_protocol(v.vpninfo, cStr) != 0 {
		return errors.New("failed to set protocol")
	}
	return nil
}

func (v *VpnInfo) SetupTunScript(script string) error {
	cStr := C.CString(script)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_setup_tun_script(v.vpninfo, cStr) != 0 {
		return errors.New("failed to set up tun script")
	}
	return nil
}

func (v *VpnInfo) SetupTunFd(fd int) error {
	if C.openconnect_setup_tun_fd(v.vpninfo, C.int(fd)) != 0 {
		return errors.New("failed to set up tun fd")
	}
	return nil
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

	if C.openconnect_setup_csd(v.vpninfo, C.uid_t(uid), silentC, cWrapper) != 0 {
		return errors.New("failed to set up CSD")
	}
	return nil
}

func (v *VpnInfo) SetupDTLS(attemptPeriod int) error {
	if C.openconnect_setup_dtls(v.vpninfo, C.int(attemptPeriod)) != 0 {
		return errors.New("failed to set up DTLS")
	}
	return nil
}

func (v *VpnInfo) DisableDTLS() error {
	if C.openconnect_disable_dtls(v.vpninfo) != 0 {
		return errors.New("failed to disable DTLS")
	}
	return nil
}

func (v *VpnInfo) SetDPD(min_seconds int) {
	C.openconnect_set_dpd(v.vpninfo, C.int(min_seconds))
}

func (v *VpnInfo) ObtainCookie() error {
	if C.openconnect_obtain_cookie(v.vpninfo) != 0 {
		return errors.New("failed to obtain auth cookie")
	}
	return nil
}

func (v *VpnInfo) SetupCmdPipe() (*CMDPipe, error) {
	fd := C.openconnect_setup_cmd_pipe(v.vpninfo)
	if fd < 0 {
		return nil, errors.New("failed to open cmd pipe")
	}
	f := os.NewFile(uintptr(fd), "")
	return &CMDPipe{w: f}, nil
}

func (v *VpnInfo) MakeCSTPConnection() error {
	if C.openconnect_make_cstp_connection(v.vpninfo) != 0 {
		return errors.New("failed to make CSTP connection")
	}
	return nil
}

func (v *VpnInfo) MainLoop() error {
	if C.openconnect_mainloop(v.vpninfo, 5, C.RECONNECT_INTERVAL_MIN) != 0 {
		return errors.New("failed to enter main loop")
	}
	return nil
}

func (v *VpnInfo) GetTunFd() (int, error) {
	return -1, errors.New("tun FD not set")
}
