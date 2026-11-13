package systemconfiguration

/*
#include <SystemConfiguration/SystemConfiguration.h>
*/
import "C"

import (
	"runtime"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
)

type NetworkService struct{ ref C.SCNetworkServiceRef }

func (prefs *Preferences) CopyService(id string) *NetworkService {
	defer runtime.KeepAlive(prefs)
	text := cf.NewString(id)
	defer C.CFRelease(C.CFTypeRef(text))
	ref := C.SCNetworkServiceCopy(prefs.ref, C.CFStringRef(text))
	if ref == 0 {
		return nil
	}
	service := &NetworkService{ref: ref}
	runtime.AddCleanup(service, func(ref C.SCNetworkServiceRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return service
}
func (service *NetworkService) Enabled() bool {
	defer runtime.KeepAlive(service)
	return C.SCNetworkServiceGetEnabled(service.ref) != 0
}
