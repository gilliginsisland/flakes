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
