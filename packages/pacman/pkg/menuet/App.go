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
		menuItems:          make(map[string]*MenuItem),
	}
})

// Application represents the singleton application instance
type Application struct {
	Name  string
	Label string

	Menu Itemer

	NotificationResponder func(NotificationResponse)

	menuItemsMu sync.RWMutex
	menuItems   map[string]*MenuItem

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
