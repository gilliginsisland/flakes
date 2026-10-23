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
	runtime.AddCleanup(url, core.Release, unsafe.Pointer(obj))
	return url
}

// WithPointer calls fn synchronously while keeping the URL alive.
// A nil URL passes a nil pointer to fn.
func (url *URL) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(url)
	var obj unsafe.Pointer
	if url != nil {
		obj = unsafe.Pointer(url.obj)
	}
	fn(obj)
}
