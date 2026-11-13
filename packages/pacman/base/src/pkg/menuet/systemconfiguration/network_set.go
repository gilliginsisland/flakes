package systemconfiguration

/*
#include <SystemConfiguration/SystemConfiguration.h>
*/
import "C"

import (
	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
	"runtime"
)

type NetworkSet struct{ ref C.SCNetworkSetRef }

func (prefs *Preferences) CopyCurrentSet() (*NetworkSet, error) {
	defer runtime.KeepAlive(prefs)
	ref := C.SCNetworkSetCopyCurrent(prefs.ref)
	if ref == 0 {
		return nil, configurationError("read current network location")
	}
	set := &NetworkSet{ref: ref}
	runtime.AddCleanup(set, func(ref C.SCNetworkSetRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return set, nil
}

// ServiceOrder returns the service IDs ordered by the current set's preferences.
func (set *NetworkSet) ServiceOrder() []string {
	defer runtime.KeepAlive(set)
	refs := C.SCNetworkSetGetServiceOrder(set.ref)
	if refs == 0 {
		return nil
	}
	ids := make([]string, 0, int(C.CFArrayGetCount(refs)))
	for index := C.CFIndex(0); index < C.CFArrayGetCount(refs); index++ {
		ids = append(ids, cf.StringFromRef(cf.StringRef(C.CFArrayGetValueAtIndex(refs, index))))
	}
	return ids
}
