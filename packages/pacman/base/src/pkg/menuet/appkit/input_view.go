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

// InputView is a menu-item view with a label, text field, and Submit/Cancel buttons.
type InputView struct {
	obj     *C.struct_MenuetInputView
	submit  *core.Callback
	cancel  *core.Callback
	results chan InputResult
}

type InputResult struct {
	Value    string
	Canceled bool
}

func NewInputView(message, placeholder string) *InputView {
	text := C.CString(message)
	defer C.free(unsafe.Pointer(text))
	hint := C.CString(placeholder)
	defer C.free(unsafe.Pointer(hint))
	view := &InputView{
		obj:     C.menuet_input_view_new(text, hint),
		results: make(chan InputResult),
	}
	runtime.AddCleanup(view, core.Release, unsafe.Pointer(view.obj))
	// The view owns its callbacks. The global callback registry must not hold
	// the view strongly through their closures and prevent its cleanup.
	target := weak.Make(view)
	view.submit = core.NewCallback(func(sender core.Sender) {
		defer core.Release(unsafe.Pointer(sender.Pointer()))
		if view := target.Value(); view != nil {
			select {
			case view.results <- InputResult{Value: view.StringValue()}:
			default:
			}
		}
	})
	view.cancel = core.NewCallback(func(sender core.Sender) {
		defer core.Release(unsafe.Pointer(sender.Pointer()))
		if view := target.Value(); view != nil {
			select {
			case view.results <- InputResult{Canceled: true}:
			default:
			}
		}
	})
	view.submit.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_input_view_set_submit_callback(view.obj, (*C.struct_MenuetCallback)(obj))
	})
	view.cancel.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_input_view_set_cancel_callback(view.obj, (*C.struct_MenuetCallback)(obj))
	})
	return view
}

// Results is unbuffered and never closed. Submit and Cancel drop their results
// when no request is waiting. The caller must keep the view alive while waiting.
func (view *InputView) Results() <-chan InputResult {
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

func (view *InputView) SetMessageText(message string) *InputView {
	defer runtime.KeepAlive(view)
	value := C.CString(message)
	defer C.free(unsafe.Pointer(value))
	C.menuet_input_view_set_message_text(view.obj, value)
	return view
}

func (view *InputView) SetStringValue(text string) *InputView {
	defer runtime.KeepAlive(view)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_input_view_set_string_value(view.obj, value)
	return view
}
