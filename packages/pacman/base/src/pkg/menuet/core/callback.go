package core

/*
#cgo CFLAGS: -x objective-c -fobjc-arc

#include "core.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

type Sender struct {
	obj *C.struct_NSObject
}

func (sender Sender) Pointer() *C.struct_NSObject {
	return sender.obj
}

// CallbackFunc owns the sender's retained reference and must release it or
// transfer it to a wrapper with cleanup.
type CallbackFunc func(Sender)

type Callback struct {
	obj *C.struct_MenuetCallback
}

var callbacks syncutil.Map[*C.struct_MenuetCallback, CallbackFunc]

func NewCallback(fn CallbackFunc) *Callback {
	callback := &Callback{obj: C.menuet_callback_new()}
	callbacks.Store(callback.obj, fn)
	runtime.AddCleanup(callback, func(obj *C.struct_MenuetCallback) {
		// Remove the mapping before release allows the native address to be reused.
		callbacks.Delete(obj)
		Release(unsafe.Pointer(obj))
	}, callback.obj)
	return callback
}

// WithPointer calls fn synchronously while keeping the callback alive.
// A nil callback passes a nil pointer to fn.
func (callback *Callback) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(callback)
	var obj unsafe.Pointer
	if callback != nil {
		obj = unsafe.Pointer(callback.obj)
	}
	fn(obj)
}

func (callback *Callback) Selector() Selector {
	return SEL("invoke:")
}

//export menuet_core_invoke_callback
func menuet_core_invoke_callback(pointer *C.struct_MenuetCallback, value *C.struct_NSObject) {
	callback, ok := callbacks.Load(pointer)
	if !ok {
		Release(unsafe.Pointer(value))
		return
	}
	callback(Sender{obj: value})
}
