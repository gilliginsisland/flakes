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
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_run(application.obj)
}

func (application *Application) Terminate() {
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_terminate(application.obj)
}

func (application *Application) ShowHelp() {
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_show_help(application.obj)
}

// SetDelegate assigns a weak Objective-C reference. The caller must keep its
// Go delegate wrapper alive while the application uses it.
func (application *Application) SetDelegate(delegate unsafe.Pointer) *Application {
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_set_delegate(application.obj, delegate)
	return application
}

func (application *Application) SetActivationPolicy(policy ActivationPolicy) *Application {
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_set_activation_policy(application.obj, C.NSApplicationActivationPolicy(policy))
	return application
}

func (application *Application) ActivateIgnoringOtherApps(flag bool) {
	defer runtime.KeepAlive(application)
	C.menuet_ns_application_activate_ignoring_other_apps(application.obj, C.bool(flag))
}
