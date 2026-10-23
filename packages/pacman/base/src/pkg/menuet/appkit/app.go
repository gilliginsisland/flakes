package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo nocallback menuet_ns_application_terminate
#cgo nocallback menuet_ns_application_set_delegate
#cgo nocallback menuet_ns_application_set_activation_policy

#import "app.h"
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Application struct {
	obj *C.struct_NSApplication
}

type ActivationPolicy int

const (
	ActivationPolicyRegular    ActivationPolicy = C.NSApplicationActivationPolicyRegular
	ActivationPolicyAccessory  ActivationPolicy = C.NSApplicationActivationPolicyAccessory
	ActivationPolicyProhibited ActivationPolicy = C.NSApplicationActivationPolicyProhibited
)

func init() {
	// AppKit's event loop must stay on the process's initial thread.
	runtime.LockOSThread()
}

var sharedApplication = sync.OnceValue(func() *Application {
	application := &Application{obj: C.menuet_ns_application_shared()}
	runtime.AddCleanup(application, core.Release, unsafe.Pointer(application.obj))
	return application
})

func SharedApplication() *Application {
	return sharedApplication()
}

// Run runs the application loop on the main goroutine.
func (application *Application) Run() {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_run((*C.struct_NSApplication)(obj))
	})
}

func (application *Application) Terminate() {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_terminate((*C.struct_NSApplication)(obj))
	})
}

func (application *Application) ShowHelp() {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_show_help((*C.struct_NSApplication)(obj))
	})
}

// SetDelegate assigns a weak Objective-C reference. The caller must keep its
// Go delegate wrapper alive while the application uses it.
func (application *Application) SetDelegate(delegate unsafe.Pointer) *Application {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_set_delegate((*C.struct_NSApplication)(obj), delegate)
	})
	return application
}

func (application *Application) SetActivationPolicy(policy ActivationPolicy) *Application {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_set_activation_policy((*C.struct_NSApplication)(obj), C.NSApplicationActivationPolicy(policy))
	})
	return application
}

func (application *Application) ActivateIgnoringOtherApps(flag bool) {
	application.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_application_activate_ignoring_other_apps((*C.struct_NSApplication)(obj), C.bool(flag))
	})
}

// WithPointer calls fn synchronously while keeping the application alive.
// A nil application passes a nil pointer to fn.
func (application *Application) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(application)
	var obj unsafe.Pointer
	if application != nil {
		obj = unsafe.Pointer(application.obj)
	}
	fn(obj)
}
