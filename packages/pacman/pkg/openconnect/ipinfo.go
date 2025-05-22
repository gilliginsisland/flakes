package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"
import "errors"

type IPInfo struct {
	Addr     string
	Netmask  string
	Addr6    string
	Netmask6 string
	DNS      []string
	NBNS     []string
	Domain   string
	ProxyPAC string
	MTU      uint32
	Gateway  string

	CSTPOptions map[string]string
	DTLSOptions map[string]string
}

func (v *VpnInfo) GetIPInfo() (*IPInfo, error) {
	var ip *C.struct_oc_ip_info
	var cstp *C.struct_oc_vpn_option
	var dtls *C.struct_oc_vpn_option

	if C.openconnect_get_ip_info(
		v.vpninfo, &ip, &cstp, &dtls,
	) != 0 {
		return nil, errors.New("failed to get IP info")
	}

	info := IPInfo{
		Addr:        C.GoString(ip.addr),
		Netmask:     C.GoString(ip.netmask),
		Addr6:       C.GoString(ip.addr6),
		Netmask6:    C.GoString(ip.netmask6),
		Domain:      C.GoString(ip.domain),
		ProxyPAC:    C.GoString(ip.proxy_pac),
		MTU:         uint32(ip.mtu),
		Gateway:     C.GoString(ip.gateway_addr),
		CSTPOptions: make(map[string]string),
		DTLSOptions: make(map[string]string),
	}

	for i := range 3 {
		if ip.dns[i] != nil {
			info.DNS = append(info.DNS, C.GoString(ip.dns[i]))
		}
		if ip.nbns[i] != nil {
			info.NBNS = append(info.NBNS, C.GoString(ip.nbns[i]))
		}
	}

	for opt := cstp; opt != nil; opt = opt.next {
		info.CSTPOptions[C.GoString(opt.option)] = C.GoString(opt.value)
	}

	for opt := dtls; opt != nil; opt = opt.next {
		info.DTLSOptions[C.GoString(opt.option)] = C.GoString(opt.value)
	}

	return &info, nil
}
