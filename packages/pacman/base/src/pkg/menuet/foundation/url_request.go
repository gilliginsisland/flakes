package foundation

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#import "url_request.h"
*/
import "C"

import (
	"runtime"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type URLRequestRef = *C.struct_NSURLRequest

type URLRequest struct {
	obj *C.struct_NSURLRequest
}

func NewURLRequest(url *URL) *URLRequest {
	defer runtime.KeepAlive(url)
	request := &URLRequest{obj: C.menuet_ns_url_request_new(url.obj)}
	runtime.AddCleanup(request, core.Release, core.ObjectRef(request.obj))
	return request
}

// WithPointer borrows the native request synchronously, keeping its owner alive.
// A nil request passes nil to fn.
func (request *URLRequest) WithPointer(fn func(URLRequestRef)) {
	defer runtime.KeepAlive(request)
	var ref URLRequestRef
	if request != nil {
		ref = request.obj
	}
	fn(ref)
}
