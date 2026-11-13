package systemconfiguration

/*
#include <SystemConfiguration/SystemConfiguration.h>
*/
import "C"

import (
	"runtime"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
)

type NetworkProtocol struct{ ref C.SCNetworkProtocolRef }

func (service *NetworkService) CopyProtocol(protocolType cf.StringRef) *NetworkProtocol {
	defer runtime.KeepAlive(service)
	ref := C.SCNetworkServiceCopyProtocol(service.ref, C.CFStringRef(protocolType))
	if ref == 0 {
		return nil
	}
	protocol := &NetworkProtocol{ref: ref}
	runtime.AddCleanup(protocol, func(ref C.SCNetworkProtocolRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return protocol
}
func (protocol *NetworkProtocol) SetEnabled(enabled bool) error {
	defer runtime.KeepAlive(protocol)
	var flag C.Boolean
	if enabled {
		flag = 1
	}
	if C.SCNetworkProtocolSetEnabled(protocol.ref, flag) == 0 {
		return configurationError("set network protocol enabled state")
	}
	return nil
}
func (protocol *NetworkProtocol) Configuration() *cf.Dictionary {
	defer runtime.KeepAlive(protocol)
	ref := cf.DictionaryRef(C.SCNetworkProtocolGetConfiguration(protocol.ref))
	return cf.OwnDictionary(cf.Retain(ref))
}
func (protocol *NetworkProtocol) SetConfiguration(config *cf.Dictionary) error {
	defer runtime.KeepAlive(protocol)
	var err error
	config.WithPointer(func(ref cf.DictionaryRef) {
		if C.SCNetworkProtocolSetConfiguration(protocol.ref, C.CFDictionaryRef(ref)) == 0 {
			err = configurationError("set network protocol configuration")
		}
	})
	return err
}
