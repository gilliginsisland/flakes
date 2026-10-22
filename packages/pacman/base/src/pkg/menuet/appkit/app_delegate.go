package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo nocallback menuet_app_delegate_new
#cgo nocallback menuet_app_delegate_set_will_finish_launching
#cgo nocallback menuet_app_delegate_set_did_finish_launching

#import "app_delegate.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type AppDelegate struct {
	obj                 *C.struct_AppDelegate
	willFinishLaunching *core.Callback
	didFinishLaunching  *core.Callback
}

func NewAppDelegate() *AppDelegate {
	delegate := &AppDelegate{obj: C.menuet_app_delegate_new()}
	runtime.AddCleanup(delegate, core.Release, unsafe.Pointer(delegate.obj))
	return delegate
}

// WithPointer calls fn synchronously while keeping the delegate alive.
// A nil delegate passes a nil pointer to fn.
func (delegate *AppDelegate) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(delegate)
	var obj unsafe.Pointer
	if delegate != nil {
		obj = unsafe.Pointer(delegate.obj)
	}
	fn(obj)
}

// SetWillFinishLaunching invokes fn synchronously on the main thread before
// launch completes, so initialization finishes before AppKit continues launching.
func (delegate *AppDelegate) SetWillFinishLaunching(fn func()) *AppDelegate {
	var callback *core.Callback
	if fn != nil {
		callback = core.NewCallback(func(sender core.Sender) {
			core.Release(unsafe.Pointer(sender.Pointer()))
			fn()
		})
	}
	delegate.WithPointer(func(obj unsafe.Pointer) {
		callback.WithPointer(func(target unsafe.Pointer) {
			C.menuet_app_delegate_set_will_finish_launching((*C.struct_AppDelegate)(obj), (*C.struct_MenuetCallback)(target))
		})
	})
	delegate.willFinishLaunching = callback
	return delegate
}

// SetDidFinishLaunching invokes fn in a goroutine after the application launches.
func (delegate *AppDelegate) SetDidFinishLaunching(fn func()) *AppDelegate {
	var callback *core.Callback
	if fn != nil {
		callback = core.NewCallback(func(sender core.Sender) {
			core.Release(unsafe.Pointer(sender.Pointer()))
			go fn()
		})
	}
	delegate.WithPointer(func(obj unsafe.Pointer) {
		callback.WithPointer(func(target unsafe.Pointer) {
			C.menuet_app_delegate_set_did_finish_launching((*C.struct_AppDelegate)(obj), (*C.struct_MenuetCallback)(target))
		})
	})
	delegate.didFinishLaunching = callback
	return delegate
}
