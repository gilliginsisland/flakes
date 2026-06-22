package openconnect

/*
#cgo pkg-config: openconnect

#cgo nocallback go_mainloop

#include <openconnect.h>
#include <stdlib.h>
#include "bridge.h"

*/
import "C"

import (
	"errors"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var handles = syncutil.Map[uintptr, *VpnInfo]{}

type Protocol string

const (
	ProtocolAnyConnect    Protocol = "anyconnect"
	ProtocolGlobalProtect Protocol = "gp"

	DefaultUserAgent = "AnyConnect-compatible OpenConnect VPN Agent"
)

type Callbacks struct {
	ValidatePeerCert   func(cert string) bool
	ProcessAuthForm    func(form *AuthForm) error
	ProcessCSD         func(info CSDInfo) error
	Progress           func(level LogLevel, message string)
	ExternalBrowser    func(uri string) error
	ReconnectedHandler func()
	ProtectSocket      func(fd int)
}

type CSDInfo struct {
	Hostname string
	SHA256   string
	Token    string
	Ticket   string
	Stub     string
}

type Options struct {
	UserAgent           string
	VersionString       string
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
	errCh   chan error
	Callbacks
}

type OpError struct {
	Op  string
	Err error
}

type DebugSnapshot struct {
	ID               string   `json:"id"`
	Hostname         string   `json:"hostname,omitempty"`
	Protocol         Protocol `json:"protocol,omitempty"`
	MainloopLoops    uint64   `json:"mainloopLoops"`
	CommandFDPolls   uint64   `json:"commandFdPolls"`
	StaleDTLSWakes   uint64   `json:"staleDtlsWakes"`
	GPSTESPFallbacks uint64   `json:"gpstEspFallbacks"`
	DTLSState        int      `json:"dtlsState"`
	DTLSStateName    string   `json:"dtlsStateName"`
	DTLSFD           int      `json:"dtlsFd"`
	SSLFD            int      `json:"sslFd"`
	TunFD            int      `json:"tunFd"`
	Timeout          int      `json:"timeout"`
	DidWork          int      `json:"didWork"`
	UDPR             int      `json:"udpR"`
	TCPR             int      `json:"tcpR"`
	TunR             int      `json:"tunR"`
	NeedPollCMDFD    int      `json:"needPollCmdFd"`
}

func (e *OpError) Error() string {
	if e.Op == "" {
		return e.Err.Error()
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *OpError) Unwrap() error {
	return e.Err
}

// NewVpnInfo initializes a new VPN session with callbacks.
func New(opts Options) (*VpnInfo, error) {
	if opts.UserAgent == "" {
		opts.UserAgent = DefaultUserAgent
	}

	cUserAgent := C.CString(opts.UserAgent)
	defer C.free(unsafe.Pointer(cUserAgent))

	vpninfo := C.go_vpninfo_new(cUserAgent, nil)
	if vpninfo == nil {
		return nil, errors.New("failed to create VPN session")
	}

	v := VpnInfo{
		vpninfo: vpninfo,
		errCh:   make(chan error, 1),
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
}

func (v *VpnInfo) ParseOpts(opts Options) error {
	v.Callbacks = opts.Callbacks

	v.SetLogLevel(opts.LogLevel)

	if opts.UserAgent != "" {
		if err := v.SetUserAgent(opts.UserAgent); err != nil {
			return err
		}
	}

	if opts.VersionString != "" {
		if err := v.SetVersionString(opts.VersionString); err != nil {
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

// SetVersionString sets the VPN client version reported in auth XML.
func (v *VpnInfo) SetVersionString(version string) error {
	cStr := C.CString(version)
	defer C.free(unsafe.Pointer(cStr))
	return ocErrno("set version string", C.openconnect_set_version_string(v.vpninfo, cStr))
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
	var enableCSD C.int
	if v.ProcessCSD != nil {
		enableCSD = 1
	}
	C.go_set_csd_callback(v.vpninfo, enableCSD)

	err := ocErrno("obtain cookie", C.openconnect_obtain_cookie(v.vpninfo))
	if err == nil {
		return nil
	}

	select {
	case callbackErr := <-v.errCh:
		return callbackErr
	default:
		return err
	}
}

func (v *VpnInfo) SetupCmdPipe() (*CMDPipe, error) {
	fd := C.openconnect_setup_cmd_pipe(v.vpninfo)
	if fd < 0 {
		return nil, ocErrno("setup cmd pipe", fd)
	}
	f := os.NewFile(uintptr(fd), "")
	return &CMDPipe{w: f, fd: int(fd)}, nil
}

func (v *VpnInfo) MakeCSTPConnection() error {
	return ocErrno("make CSTP connection", C.openconnect_make_cstp_connection(v.vpninfo))
}

func (v *VpnInfo) MainLoop() error {
	return ocErrno("main loop", C.go_mainloop(v.vpninfo))
}

func (v *VpnInfo) GetTunFd() (int, error) {
	var err error
	fd := int(C.openconnect_get_tun_fd(v.vpninfo))
	if fd < 0 {
		err = errors.New("get tun fd: fd not setup")
	}
	return fd, err
}

func DebugSnapshots() []DebugSnapshot {
	var snapshots []DebugSnapshot
	handles.Range(func(_, value any) bool {
		v := value.(*VpnInfo)
		if snapshot, ok := v.DebugSnapshot(); ok {
			snapshots = append(snapshots, snapshot)
		}
		return true
	})
	return snapshots
}

func (v *VpnInfo) DebugSnapshot() (DebugSnapshot, bool) {
	if v == nil || v.vpninfo == nil {
		return DebugSnapshot{}, false
	}

	var debug C.struct_openconnect_pacman_debug
	if C.openconnect_get_pacman_debug(v.vpninfo, &debug) != 0 {
		return DebugSnapshot{}, false
	}

	dtlsState := int(debug.dtls_state)
	return DebugSnapshot{
		ID:               strconv.FormatUint(uint64(uintptr(unsafe.Pointer(v.vpninfo))), 16),
		Hostname:         v.Hostname(),
		Protocol:         v.Protocol(),
		MainloopLoops:    uint64(debug.mainloop_loops),
		CommandFDPolls:   uint64(debug.command_fd_polls),
		StaleDTLSWakes:   uint64(debug.stale_dtls_wakes),
		GPSTESPFallbacks: uint64(debug.gpst_esp_fallbacks),
		DTLSState:        dtlsState,
		DTLSStateName:    dtlsStateName(dtlsState),
		DTLSFD:           int(debug.dtls_fd),
		SSLFD:            int(debug.ssl_fd),
		TunFD:            int(debug.tun_fd),
		Timeout:          int(debug.timeout),
		DidWork:          int(debug.did_work),
		UDPR:             int(debug.udp_r),
		TCPR:             int(debug.tcp_r),
		TunR:             int(debug.tun_r),
		NeedPollCMDFD:    int(debug.need_poll_cmd_fd),
	}, true
}

func dtlsStateName(state int) string {
	switch state {
	case 0:
		return "NOSECRET"
	case 1:
		return "SECRET"
	case 2:
		return "DISABLED"
	case 3:
		return "SLEEPING"
	case 4:
		return "CONNECTING"
	case 5:
		return "CONNECTED"
	case 6:
		return "ESTABLISHED"
	default:
		return "UNKNOWN"
	}
}

func ocErrno(op string, rc C.int) error {
	if rc == 0 {
		return nil
	}
	return &OpError{
		Op:  op,
		Err: syscall.Errno(-rc),
	}
}
