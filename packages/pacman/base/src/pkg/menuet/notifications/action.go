package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "action.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type ActionOptions uint64

const (
	ActionOptionNone       ActionOptions = 0
	ActionOptionForeground ActionOptions = C.UNNotificationActionOptionForeground
)

type Action struct {
	obj *C.struct_UNNotificationAction
}

type ActionContent struct {
	Identifier string
	Title      string
	Options    ActionOptions
}

type TextInputActionContent struct {
	Identifier           string
	Title                string
	Options              ActionOptions
	TextInputButtonTitle string
	TextInputPlaceholder string
}

func newAction(obj *C.struct_UNNotificationAction) *Action {
	action := &Action{obj: obj}
	runtime.AddCleanup(action, core.Release, unsafe.Pointer(obj))
	return action
}

func NewAction(content ActionContent) *Action {
	id := C.CString(content.Identifier)
	defer C.free(unsafe.Pointer(id))
	text := C.CString(content.Title)
	defer C.free(unsafe.Pointer(text))
	return newAction(C.menuet_un_action_new(id, text, C.UNNotificationActionOptions(content.Options)))
}

// NewTextInputAction returns an action whose native instance is a
// UNTextInputNotificationAction, a subclass of UNNotificationAction.
func NewTextInputAction(content TextInputActionContent) *Action {
	id := C.CString(content.Identifier)
	defer C.free(unsafe.Pointer(id))
	text := C.CString(content.Title)
	defer C.free(unsafe.Pointer(text))
	submit := C.CString(content.TextInputButtonTitle)
	defer C.free(unsafe.Pointer(submit))
	hint := C.CString(content.TextInputPlaceholder)
	defer C.free(unsafe.Pointer(hint))
	return newAction(C.menuet_un_text_input_action_new(id, text, C.UNNotificationActionOptions(content.Options), submit, hint))
}

// WithPointer calls fn synchronously while keeping the action alive.
// A nil action passes a nil pointer to fn.
func (action *Action) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(action)
	var obj unsafe.Pointer
	if action != nil {
		obj = unsafe.Pointer(action.obj)
	}
	fn(obj)
}
