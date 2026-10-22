package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

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

var sharedApplication = sync.OnceValue(func() *Application {
	application := &Application{obj: C.menuet_ns_application_shared()}
	runtime.AddCleanup(application, core.Release, unsafe.Pointer(application.obj))
	return application
})

func SharedApplication() *Application {
	return sharedApplication()
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
