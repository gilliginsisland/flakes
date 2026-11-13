package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#import "switch.h"
*/
import "C"

import (
	"runtime"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type SwitchRef = *C.struct_NSSwitch

// NSSwitch owns a native switch, independently of its containing menu item.
type NSSwitch struct {
	obj *C.struct_NSSwitch
}

func newNSSwitch(ref SwitchRef) *NSSwitch {
	control := &NSSwitch{obj: ref}
	runtime.AddCleanup(control, core.Release, core.ObjectRef(ref))
	return control
}

func (control *NSSwitch) WithPointer(fn func(SwitchRef)) {
	defer runtime.KeepAlive(control)
	var ref SwitchRef
	if control != nil {
		ref = control.obj
	}
	fn(ref)
}

func (control *NSSwitch) State() ControlStateValue {
	defer runtime.KeepAlive(control)
	return ControlStateValue(C.menuet_ns_switch_state(control.obj))
}

func (control *NSSwitch) SetState(state ControlStateValue) {
	defer runtime.KeepAlive(control)
	C.menuet_ns_switch_set_state(control.obj, C.NSInteger(state))
}

func (control *NSSwitch) Enabled() bool {
	defer runtime.KeepAlive(control)
	return bool(C.menuet_ns_switch_enabled(control.obj))
}

func (control *NSSwitch) SetEnabled(enabled bool) {
	defer runtime.KeepAlive(control)
	C.menuet_ns_switch_set_enabled(control.obj, C.bool(enabled))
}

// SetSpinning controls the menu row's progress overlay independently of enabled.
func (control *NSSwitch) SetSpinning(spinning bool) {
	defer runtime.KeepAlive(control)
	C.menuet_ns_switch_set_spinning(control.obj, C.bool(spinning))
}

// SetCallback assigns the native target and action. The caller keeps it alive.
func (control *NSSwitch) SetCallback(callback *core.Callback) {
	defer runtime.KeepAlive(control)
	var action core.Selector
	if callback != nil {
		action = callback.Selector()
	}
	callback.WithPointer(func(ref core.CallbackRef) {
		C.menuet_ns_switch_set_callback(control.obj, (*C.struct_MenuetCallback)(ref), (*C.struct_objc_selector)(action.Pointer()))
	})
}

// Menu returns the containing menu, or nil when the switch is unattached.
func (control *NSSwitch) Menu() *Menu {
	defer runtime.KeepAlive(control)
	return newMenu(C.menuet_ns_switch_menu(control.obj))
}
