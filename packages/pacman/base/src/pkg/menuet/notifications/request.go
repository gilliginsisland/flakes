package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "request.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Request struct {
	obj *C.struct_UNNotificationRequest
}

type RequestOptions struct {
	Identifier string
	Content    *Content
}

func NewRequest(options RequestOptions) *Request {
	defer runtime.KeepAlive(options.Content)
	value := C.CString(options.Identifier)
	defer C.free(unsafe.Pointer(value))
	request := &Request{obj: C.menuet_un_request_new(value, options.Content.obj)}
	runtime.AddCleanup(request, core.Release, unsafe.Pointer(request.obj))
	return request
}
