package openconnect

/*
#include <openconnect.h>
#include <stdlib.h>
*/
import "C"

import "unsafe"

// DeviceIdentity describes the local endpoint in AnyConnect authentication XML.
type DeviceIdentity struct {
	PlatformVersion string
	DeviceType      string
	UniqueID        string
}

func (v *VpnInfo) SetDeviceIdentity(identity DeviceIdentity) error {
	native := [3]*C.char{
		C.CString(identity.PlatformVersion),
		C.CString(identity.DeviceType),
		C.CString(identity.UniqueID),
	}
	for _, value := range native {
		defer C.free(unsafe.Pointer(value))
	}
	return ocErrno("set device identity", C.openconnect_set_mobile_info(v.vpninfo, native[0], native[1], native[2]))
}
