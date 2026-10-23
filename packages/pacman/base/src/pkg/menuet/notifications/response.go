package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "response.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

var (
	DefaultActionIdentifier = stringValue(C.menuet_un_default_action_identifier())
	DismissActionIdentifier = stringValue(C.menuet_un_dismiss_action_identifier())
)

type Response struct {
	obj *C.struct_UNNotificationResponse
}

func newResponse(obj *C.struct_UNNotificationResponse) *Response {
	response := &Response{obj: obj}
	runtime.AddCleanup(response, core.Release, unsafe.Pointer(obj))
	return response
}

func stringValue(value *C.char) string {
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func (response *Response) NotificationIdentifier() string {
	defer runtime.KeepAlive(response)
	return stringValue(C.menuet_un_response_notification_identifier(response.obj))
}

func (response *Response) ActionIdentifier() string {
	defer runtime.KeepAlive(response)
	return stringValue(C.menuet_un_response_action_identifier(response.obj))
}

func (response *Response) Text() string {
	defer runtime.KeepAlive(response)
	return stringValue(C.menuet_un_response_text(response.obj))
}

// WithPointer calls fn synchronously while keeping the response alive.
// A nil response passes a nil pointer to fn.
func (response *Response) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(response)
	var obj unsafe.Pointer
	if response != nil {
		obj = unsafe.Pointer(response.obj)
	}
	fn(obj)
}
