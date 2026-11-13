package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#import "alert.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type ModalResponse int

const (
	AlertFirstButtonReturn  ModalResponse = C.NSAlertFirstButtonReturn
	AlertSecondButtonReturn ModalResponse = C.NSAlertSecondButtonReturn
)

type Alert struct {
	obj *C.struct_NSAlert
}

func NewAlert() *Alert {
	alert := &Alert{obj: C.menuet_ns_alert_new()}
	runtime.AddCleanup(alert, core.Release, core.ObjectRef(alert.obj))
	return alert
}

func (alert *Alert) SetMessageText(text string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_set_message_text(alert.obj, value)
	return alert
}

func (alert *Alert) SetInformativeText(text string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_set_informative_text(alert.obj, value)
	return alert
}

func (alert *Alert) AddButtonWithTitle(title string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_add_button_with_title(alert.obj, value)
	return alert
}

// RunModal presents an application-modal alert and waits for a button response.
func (alert *Alert) RunModal() ModalResponse {
	defer runtime.KeepAlive(alert)
	return ModalResponse(C.menuet_ns_alert_run_modal(alert.obj))
}
