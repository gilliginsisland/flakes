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

var callbacks syncutil.Map[uintptr, CallbackFunc]

func NewCallback(fn CallbackFunc) *Callback {
	callback := &Callback{obj: C.menuet_callback_new()}
	pointer := unsafe.Pointer(callback.obj)
	callbacks.Store(uintptr(pointer), fn)
	runtime.AddCleanup(callback, func(pointer unsafe.Pointer) {
		// Remove the mapping before release allows the native address to be reused.
		callbacks.Delete(uintptr(pointer))
		Release(pointer)
	}, pointer)
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
func menuet_core_invoke_callback(pointer uintptr, value unsafe.Pointer) {
	callback, ok := callbacks.Load(pointer)
	if !ok {
		Release(value)
		return
	}
	callback(Sender{obj: (*C.struct_NSObject)(value)})
}
