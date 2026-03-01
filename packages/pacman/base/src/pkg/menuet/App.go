package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#cgo nocallback invoke_app_action
#cgo nocallback has_app_action
#cgo nocallback terminateApplication

#import "AppDelegate.h"

extern void invoke_app_action(const char *actionKey, void *data);
extern int has_app_action(const char *actionKey);
*/
import "C"

import (
	"sync"
	"unsafe"
)

// App returns the application singleton
var App = sync.OnceValue(func() *Application {
	return &Application{
		didFinishLaunching: make(chan struct{}),
	}
})

// Application represents the singleton application instance
type Application struct {
	NotificationResponder func(NotificationResponse)
	didFinishLaunching    chan struct{}
}

// AppAction interface for defining app actions with action key and data pointer
type AppAction interface {
	Action() string       // Returns the action key for the app action
	Data() unsafe.Pointer // Returns the data as a C void pointer
}

// InvokeAction invokes an app action with the specified key and data
func (app *Application) InvokeAction(action AppAction) {
	cAction := C.CString(action.Action())
	defer C.free(unsafe.Pointer(cAction))
	C.invoke_app_action(cAction, action.Data())
}

// HasAction checks if an app action exists for the given key
func (app *Application) HasAction(action AppAction) bool {
	cAction := C.CString(action.Action())
	defer C.free(unsafe.Pointer(cAction))
	return C.has_app_action(cAction) != 0
}

func (app *Application) Run(f func()) {
	go func() {
		<-app.didFinishLaunching
		f()
	}()
	C.runApplication()
}

func (app *Application) Terminate() {
	C.terminateApplication()
}

//export goAppWillFinishLaunching
func goAppWillFinishLaunching() {}

//export goAppDidFinishLaunching
func goAppDidFinishLaunching() {
	go close(App().didFinishLaunching)
}

//export goAppWillTerminate
func goAppWillTerminate() {}
