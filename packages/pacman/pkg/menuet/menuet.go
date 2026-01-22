package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

#import "menuet.h"

*/
import "C"

import (
	"encoding/json"
	"log"
	"unsafe"
)

// SetMenuState changes what is shown in the status bar
func (a *Application) SetMenuState(state *MenuState) {
	titleStr := C.CString(string(state.Title))
	imageStr := C.CString(string(state.Image))
	defer C.free(unsafe.Pointer(titleStr))
	defer C.free(unsafe.Pointer(imageStr))
	C.setState(titleStr, imageStr)
}

// MenuChanged refreshes any open menus
func (a *Application) MenuChanged() {
	C.menuChanged()
}

// MenuState represents the title and drop down,
type MenuState struct {
	Title string
	Image string // // In Resources dir or URL, should have height 22
}

func (a *Application) clicked(unique string) {
	a.visibleMenuItemsMutex.RLock()
	item, ok := a.visibleMenuItems[unique]
	a.visibleMenuItemsMutex.RUnlock()
	if !ok {
		log.Printf("Item not found for click: %s", unique)
	}
	if item.Clicked != nil {
		go item.Clicked()
	}
}

//export itemClicked
func itemClicked(uniqueCString *C.char) {
	unique := C.GoString(uniqueCString)
	App().clicked(unique)
}

//export children
func children(uniqueCString *C.char) *C.char {
	unique := C.GoString(uniqueCString)
	items := App().children(unique)
	if items == nil {
		return nil
	}
	b, err := json.Marshal(items)
	if err != nil {
		log.Printf("Marshal: %v", err)
		return nil
	}
	return C.CString(string(b))
}

//export menuClosed
func menuClosed(uniqueCString *C.char) {
	unique := C.GoString(uniqueCString)
	App().menuClosed(unique)
}
