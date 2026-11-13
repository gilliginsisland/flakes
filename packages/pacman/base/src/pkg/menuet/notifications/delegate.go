package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#import "delegate.h"
#import "response.h"
*/
import "C"

import (
	"runtime"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type DelegateRef = *C.struct_MenuetNotificationDelegate

type Delegate struct {
	obj      *C.struct_MenuetNotificationDelegate
	response *core.Callback
}

func NewDelegate() *Delegate {
	delegate := &Delegate{obj: C.menuet_un_delegate_new()}
	runtime.AddCleanup(delegate, core.Release, core.ObjectRef(delegate.obj))
	return delegate
}

// SetDidReceiveResponse invokes fn in a goroutine with an owned response.
func (delegate *Delegate) SetDidReceiveResponse(fn func(*Response)) *Delegate {
	defer runtime.KeepAlive(delegate)
	var callback *core.Callback
	if fn != nil {
		callback = core.NewCallback(func(sender core.Sender) {
			response := newResponse((*C.struct_UNNotificationResponse)(sender.Pointer()))
			go fn(response)
		})
	}
	callback.WithPointer(func(obj core.CallbackRef) {
		C.menuet_un_delegate_set_response_callback(delegate.obj, (*C.struct_MenuetCallback)(obj))
	})
	delegate.response = callback
	return delegate
}

// WithPointer calls fn synchronously while keeping the delegate alive.
// A nil delegate passes a nil pointer to fn.
func (delegate *Delegate) WithPointer(fn func(DelegateRef)) {
	defer runtime.KeepAlive(delegate)
	var obj DelegateRef
	if delegate != nil {
		obj = delegate.obj
	}
	fn(obj)
}
