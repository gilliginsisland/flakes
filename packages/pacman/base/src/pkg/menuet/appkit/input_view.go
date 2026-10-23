package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "input_view.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
	"weak"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

// InputView is an editable menu-item text field that submits on Enter.
type InputView struct {
	obj     *C.struct_MenuetInputView
	submit  *core.Callback
	results chan string
}

func NewInputView(placeholder string) *InputView {
	hint := C.CString(placeholder)
	defer C.free(unsafe.Pointer(hint))
	view := &InputView{
		obj:     C.menuet_input_view_new(hint),
		results: make(chan string),
	}
	runtime.AddCleanup(view, core.Release, unsafe.Pointer(view.obj))
	// The view owns its callback. The global callback registry must not hold
	// the view strongly through the closure and prevent its cleanup.
	target := weak.Make(view)
	view.submit = core.NewCallback(func(sender core.Sender) {
		defer core.Release(unsafe.Pointer(sender.Pointer()))
		if view := target.Value(); view != nil {
			select {
			case view.results <- view.StringValue():
			default:
			}
		}
	})
	view.submit.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_input_view_set_submit_callback(view.obj, (*C.struct_MenuetCallback)(obj))
	})
	return view
}

// Results is unbuffered and never closed. Submissions are dropped when no
// request is waiting. The caller must keep the view alive while waiting.
func (view *InputView) Results() <-chan string {
	return view.results
}

// WithPointer calls fn synchronously while keeping the view alive.
// A nil view passes a nil pointer to fn.
func (view *InputView) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(view)
	var obj unsafe.Pointer
	if view != nil {
		obj = unsafe.Pointer(view.obj)
	}
	fn(obj)
}

func (view *InputView) StringValue() string {
	defer runtime.KeepAlive(view)
	value := C.menuet_input_view_string_value(view.obj)
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func (view *InputView) SetStringValue(text string) *InputView {
	defer runtime.KeepAlive(view)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_input_view_set_string_value(view.obj, value)
	return view
}
