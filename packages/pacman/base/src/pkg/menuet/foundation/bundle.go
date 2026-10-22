package foundation

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>
#import "bundle.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Bundle struct {
	obj *C.struct_NSBundle
}

func newBundle(obj *C.struct_NSBundle) *Bundle {
	if obj == nil {
		return nil
	}
	bundle := &Bundle{obj: obj}
	runtime.AddCleanup(bundle, core.Release, unsafe.Pointer(obj))
	return bundle
}

var MainBundle = sync.OnceValue(func() *Bundle {
	return newBundle(C.menuet_ns_bundle_main())
})

func BundleWithPath(path string) *Bundle {
	value := C.CString(path)
	defer C.free(unsafe.Pointer(value))
	return newBundle(C.menuet_ns_bundle_with_path(value))
}

func (bundle *Bundle) BuiltInPlugInsPath() string {
	defer runtime.KeepAlive(bundle)
	value := C.menuet_ns_bundle_built_in_plugins_path(bundle.obj)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func (bundle *Bundle) Load() error {
	defer runtime.KeepAlive(bundle)
	var message *C.char
	if !C.menuet_ns_bundle_load(bundle.obj, &message) {
		defer C.free(unsafe.Pointer(message))
		return errors.New(C.GoString(message))
	}
	return nil
}

// PrincipalClass returns the runtime class, not an owned instance.
// Loaded bundles are not unloaded when their Go wrappers are collected.
func (bundle *Bundle) PrincipalClass() unsafe.Pointer {
	defer runtime.KeepAlive(bundle)
	return C.menuet_ns_bundle_principal_class(bundle.obj)
}

// WithPointer calls fn synchronously while keeping the bundle alive.
// A nil bundle passes a nil pointer to fn.
func (bundle *Bundle) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(bundle)
	var obj unsafe.Pointer
	if bundle != nil {
		obj = unsafe.Pointer(bundle.obj)
	}
	fn(obj)
}
