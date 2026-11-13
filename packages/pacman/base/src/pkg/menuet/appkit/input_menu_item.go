package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "input_menu_item.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
	"weak"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

// InputMenuItem owns an editable menu row that submits on Enter.
type InputMenuItem struct {
	*MenuItem
	submit  *core.Callback
	results chan string
}

func NewInputMenuItem(placeholder string) *InputMenuItem {
	hint := C.CString(placeholder)
	defer C.free(unsafe.Pointer(hint))
	item := &InputMenuItem{
		MenuItem: newMenuItem((*C.struct_NSMenuItem)(C.menuet_input_menu_item_new(hint))),
		results:  make(chan string),
	}
	// The item owns its callback. The global callback registry must not hold
	// the item strongly through the closure and prevent its cleanup.
	target := weak.Make(item)
	item.submit = core.NewCallback(func(sender core.Sender) {
		defer core.Release(sender.Pointer())
		if item := target.Value(); item != nil {
			select {
			case item.results <- item.StringValue():
			default:
			}
		}
	})
	item.submit.WithPointer(func(obj core.CallbackRef) {
		C.menuet_input_menu_item_set_submit_callback((*C.struct_MenuetInputMenuItem)(item.obj), (*C.struct_MenuetCallback)(obj))
	})
	return item
}

// Results is unbuffered and never closed. Submissions are dropped when no
// request is waiting. The caller must keep the item alive while waiting.
func (item *InputMenuItem) Results() <-chan string {
	return item.results
}

func (item *InputMenuItem) StringValue() string {
	defer runtime.KeepAlive(item)
	value := C.menuet_input_menu_item_string_value((*C.struct_MenuetInputMenuItem)(item.obj))
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}

func (item *InputMenuItem) SetStringValue(text string) *InputMenuItem {
	defer runtime.KeepAlive(item)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_input_menu_item_set_string_value((*C.struct_MenuetInputMenuItem)(item.obj), value)
	return item
}
