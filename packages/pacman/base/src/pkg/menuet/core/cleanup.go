package core

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include "core.h"
*/
import "C"

import "unsafe"

// ObjectRef is a borrowed or owned opaque NSObject reference, as documented by its API.
type ObjectRef = *C.struct_NSObject

// Release relinquishes one owned Objective-C reference.
func Release(object ObjectRef) {
	C.objc_release(unsafe.Pointer(object))
}
