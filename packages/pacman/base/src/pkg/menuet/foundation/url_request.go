package foundation

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#import "url_request.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type URLRequest struct {
	obj *C.struct_NSURLRequest
}

func NewURLRequest(url *URL) *URLRequest {
	defer runtime.KeepAlive(url)
	request := &URLRequest{obj: C.menuet_ns_url_request_new(url.obj)}
	runtime.AddCleanup(request, core.Release, unsafe.Pointer(request.obj))
	return request
}

// WithPointer calls fn synchronously while keeping the request alive.
// A nil request passes a nil pointer to fn.
func (request *URLRequest) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(request)
	var obj unsafe.Pointer
	if request != nil {
		obj = unsafe.Pointer(request.obj)
	}
	fn(obj)
}
