package core

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include "core.h"
*/
import "C"

import "unsafe"

// Release relinquishes one owned Objective-C reference.
func Release(object unsafe.Pointer) {
	C.objc_release(object)
}
