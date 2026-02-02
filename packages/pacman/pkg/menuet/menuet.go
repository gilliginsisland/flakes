package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

#import "menuet.h"

*/
import "C"

import (
	"log"
	"unsafe"
)

// SetMenuState changes what is shown in the status bar
func (a *Application) SetMenuState(state *MenuState) {
	titleStr := C.CString(string(state.Title))
	imageStr := C.CString(string(state.Image))
	defer C.free(unsafe.Pointer(titleStr))
	defer C.free(unsafe.Pointer(imageStr))
	C.set_state(titleStr, imageStr)
}

// MenuChanged refreshes any open menus
func (a *Application) MenuChanged() {
	if a.Menu == nil {
		return
	}
	a.menuItemsMu.Lock()
	defer a.menuItemsMu.Unlock()
	C.menu_changed(a.Menu.item())
}

// MenuState represents the title and drop down,
type MenuState struct {
	Title string
	Image string // // In Resources dir or URL, should have height 22
}

func (a *Application) clicked(unique string) {
	a.menuItemsMu.RLock()
	item := menuitems.Load(unique)
	a.menuItemsMu.RUnlock()
	if item == nil {
		log.Printf("Item not found for click: %s", unique)
	} else if item.Clicked != nil {
		go item.Clicked()
	}
}

//export goItemClicked
func goItemClicked(unique *C.char) {
	App().clicked(C.GoString(unique))
}
