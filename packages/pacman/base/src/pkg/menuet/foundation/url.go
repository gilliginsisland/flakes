package foundation

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#include <stdlib.h>
#import "url.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type URLRef = *C.struct_NSURL

type URL struct {
	obj *C.struct_NSURL
}

func URLWithString(value string) *URL {
	text := C.CString(value)
	defer C.free(unsafe.Pointer(text))
	obj := C.menuet_ns_url_with_string(text)
	if obj == nil {
		return nil
	}
	url := &URL{obj: obj}
	runtime.AddCleanup(url, core.Release, core.ObjectRef(obj))
	return url
}

// WithPointer borrows the native URL synchronously, keeping its owner alive.
// A nil URL passes nil to fn.
func (url *URL) WithPointer(fn func(URLRef)) {
	defer runtime.KeepAlive(url)
	var ref URLRef
	if url != nil {
		ref = url.obj
	}
	fn(ref)
}
