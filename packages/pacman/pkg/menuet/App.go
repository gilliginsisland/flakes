package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "AppDelegate.h"

*/
import "C"

import "sync"

// App returns the application singleton
var App = sync.OnceValue(func() *Application {
	return &Application{
		didFinishLaunching: make(chan struct{}),
		visibleMenuItems:   make(map[string]internalItem),
	}
})

// Application represents the singleton application instance
type Application struct {
	Name  string
	Label string

	// Children returns the top level children
	Children func() []MenuItem

	NotificationResponder func(NotificationResponse)

	currentState          *MenuState
	nextState             *MenuState
	pendingStateChange    bool
	debounceMutex         sync.Mutex
	visibleMenuItemsMutex sync.RWMutex
	visibleMenuItems      map[string]internalItem

	didFinishLaunching chan struct{}
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
	close(App().didFinishLaunching)
}

//export goAppWillTerminate
func goAppWillTerminate() {}
