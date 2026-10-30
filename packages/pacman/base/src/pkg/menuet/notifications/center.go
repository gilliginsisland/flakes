package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "center.h"
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type AuthorizationOptions uint64

const (
	AuthorizationOptionAlert AuthorizationOptions = C.UNAuthorizationOptionAlert
	AuthorizationOptionSound AuthorizationOptions = C.UNAuthorizationOptionSound
	AuthorizationOptionBadge AuthorizationOptions = C.UNAuthorizationOptionBadge
)

type Center struct {
	obj *C.struct_UNUserNotificationCenter
}

var CurrentCenter = sync.OnceValue(func() *Center {
	center := &Center{obj: C.menuet_un_center_current()}
	runtime.AddCleanup(center, core.Release, unsafe.Pointer(center.obj))
	return center
})

// SetDelegate assigns a weak native reference. Keep the Go delegate alive and
// install it before the application finishes launching.
func (center *Center) SetDelegate(delegate unsafe.Pointer) *Center {
	defer runtime.KeepAlive(center)
	C.menuet_un_center_set_delegate(center.obj, delegate)
	return center
}

func (center *Center) SetCategories(categories []*Category) *Center {
	defer runtime.KeepAlive(center)
	defer runtime.KeepAlive(categories)
	values := make([]*C.struct_UNNotificationCategory, len(categories))
	for index, category := range categories {
		values[index] = category.obj
	}
	C.menuet_un_center_set_categories(center.obj, unsafe.SliceData(values), C.size_t(len(values)))
	return center
}

// RequestAuthorization starts the native asynchronous authorization request.
// Errors and denial are reported to the system log.
func (center *Center) RequestAuthorization(options AuthorizationOptions) {
	defer runtime.KeepAlive(center)
	C.menuet_un_center_request_authorization(center.obj, C.UNAuthorizationOptions(options))
}

// AddRequest submits the notification asynchronously. Delivery errors are
// reported to the system log.
func (center *Center) AddRequest(request *Request) {
	defer runtime.KeepAlive(center)
	defer runtime.KeepAlive(request)
	C.menuet_un_center_add_request(center.obj, request.obj)
}

func (center *Center) RemovePendingRequests(identifier string) {
	defer runtime.KeepAlive(center)
	value := C.CString(identifier)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_center_remove_pending_requests(center.obj, value)
}

func (center *Center) RemoveDeliveredNotifications(identifier string) {
	defer runtime.KeepAlive(center)
	value := C.CString(identifier)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_center_remove_delivered_notifications(center.obj, value)
}

func (center *Center) RemoveAllPendingRequests() {
	defer runtime.KeepAlive(center)
	C.menuet_un_center_remove_all_pending_requests(center.obj)
}

func (center *Center) RemoveAllDeliveredNotifications() {
	defer runtime.KeepAlive(center)
	C.menuet_un_center_remove_all_delivered_notifications(center.obj)
}

// WithPointer calls fn synchronously while keeping the center alive.
// A nil center passes a nil pointer to fn.
func (center *Center) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(center)
	var obj unsafe.Pointer
	if center != nil {
		obj = unsafe.Pointer(center.obj)
	}
	fn(obj)
}
