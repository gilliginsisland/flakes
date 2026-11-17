package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#import "switch_menu_item.h"
*/
import "C"

import "unsafe"

// SwitchMenuItem is a menu header with a bold title, optional inline subtitle,
// native switch, and spinner. SetSubtitle updates the secondary text.
// Its menu-item state and embedded switch state are independent.
type SwitchMenuItem struct {
	*MenuItem
	Switch *NSSwitch
}

func NewSwitchMenuItem(title string) *SwitchMenuItem {
	text := C.CString(title)
	defer C.free(unsafe.Pointer(text))
	ref := C.menuet_switch_menu_item_new(text)
	return &SwitchMenuItem{
		MenuItem: newMenuItem((*C.struct_NSMenuItem)(ref)),
		Switch:   newNSSwitch(C.menuet_switch_menu_item_switch(ref)),
	}
}
