package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "../core/core.h"
#import "menu.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Menu struct {
	obj *C.struct_NSMenu
}

type MenuItem struct {
	obj *C.struct_NSMenuItem
}

type StatusItem struct {
	obj *C.struct_NSStatusItem
}

func NewMenu() *Menu {
	menu := &Menu{obj: C.menuet_ns_menu_new()}
	runtime.AddCleanup(menu, core.Release, unsafe.Pointer(menu.obj))
	return menu
}

func NewMenuItem() *MenuItem {
	return newMenuItem(C.menuet_ns_menu_item_new())
}

func NewMenuItemSeparator() *MenuItem {
	return newMenuItem(C.menuet_ns_menu_item_separator())
}

func NewMenuItemSectionHeader(title string) *MenuItem {
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	return newMenuItem(C.menuet_ns_menu_item_section_header(value))
}

func newMenuItem(obj *C.struct_NSMenuItem) *MenuItem {
	item := &MenuItem{obj: obj}
	runtime.AddCleanup(item, core.Release, unsafe.Pointer(item.obj))
	return item
}

func NewStatusItem() *StatusItem {
	item := &StatusItem{obj: C.menuet_ns_status_item_new()}
	runtime.AddCleanup(item, core.Release, unsafe.Pointer(item.obj))
	return item
}

// WithPointer calls fn synchronously while keeping the menu alive.
// A nil menu passes a nil pointer to fn.
func (menu *Menu) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(menu)
	var obj unsafe.Pointer
	if menu != nil {
		obj = unsafe.Pointer(menu.obj)
	}
	fn(obj)
}

// WithPointer calls fn synchronously while keeping the menu item alive.
// A nil menu item passes a nil pointer to fn.
func (item *MenuItem) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(item)
	var obj unsafe.Pointer
	if item != nil {
		obj = unsafe.Pointer(item.obj)
	}
	fn(obj)
}

// WithPointer calls fn synchronously while keeping the status item alive.
// A nil status item passes a nil pointer to fn.
func (item *StatusItem) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(item)
	var obj unsafe.Pointer
	if item != nil {
		obj = unsafe.Pointer(item.obj)
	}
	fn(obj)
}

func (menu *Menu) Insert(item *MenuItem, index int) {
	menu.WithPointer(func(menuID unsafe.Pointer) {
		item.WithPointer(func(itemID unsafe.Pointer) {
			C.menuet_ns_menu_insert((*C.struct_NSMenu)(menuID), (*C.struct_NSMenuItem)(itemID), C.int(index))
		})
	})
}

func (menu *Menu) Append(items ...*MenuItem) *Menu {
	menu.WithPointer(func(menuID unsafe.Pointer) {
		for _, item := range items {
			item.WithPointer(func(itemID unsafe.Pointer) {
				C.menuet_ns_menu_append((*C.struct_NSMenu)(menuID), (*C.struct_NSMenuItem)(itemID))
			})
		}
	})
	return menu
}

func (menu *Menu) Remove(item *MenuItem) {
	menu.WithPointer(func(menuID unsafe.Pointer) {
		item.WithPointer(func(itemID unsafe.Pointer) {
			C.menuet_ns_menu_remove((*C.struct_NSMenu)(menuID), (*C.struct_NSMenuItem)(itemID))
		})
	})
}

func (menu *Menu) Items() []*MenuItem {
	var values C.menuet_ns_menu_item_array
	menu.WithPointer(func(obj unsafe.Pointer) {
		values = C.menuet_ns_menu_items((*C.struct_NSMenu)(obj))
	})
	defer C.free(unsafe.Pointer(values.items))
	items := make([]*MenuItem, values.count)
	for index, value := range unsafe.Slice(values.items, values.count) {
		items[index] = newMenuItem(value)
	}
	return items
}

func (item *MenuItem) SetTitle(title string) *MenuItem {
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_title((*C.struct_NSMenuItem)(obj), value)
	})
	return item
}

func (item *MenuItem) SetBadge(badge string) *MenuItem {
	value := C.CString(badge)
	defer C.free(unsafe.Pointer(value))
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_badge((*C.struct_NSMenuItem)(obj), value)
	})
	return item
}

func (item *MenuItem) SetHidden(hidden bool) *MenuItem {
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_hidden((*C.struct_NSMenuItem)(obj), C.bool(hidden))
	})
	return item
}

// RepresentedObject returns an owned Objective-C reference. The caller must
// release it or transfer it to a wrapper with cleanup.
func (item *MenuItem) RepresentedObject() unsafe.Pointer {
	var object unsafe.Pointer
	item.WithPointer(func(obj unsafe.Pointer) {
		object = C.menuet_ns_menu_item_represented_object((*C.struct_NSMenuItem)(obj))
	})
	return object
}

// SetRepresentedObject sets the Objective-C object retained by the menu item.
// object must be an Objective-C object pointer or nil, not a Go pointer.
func (item *MenuItem) SetRepresentedObject(object unsafe.Pointer) *MenuItem {
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_represented_object((*C.struct_NSMenuItem)(obj), object)
	})
	return item
}

func (item *MenuItem) Tag() int {
	var tag int
	item.WithPointer(func(obj unsafe.Pointer) {
		tag = int(C.menuet_ns_menu_item_tag((*C.struct_NSMenuItem)(obj)))
	})
	return tag
}

func (item *MenuItem) SetTag(tag int) *MenuItem {
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_tag((*C.struct_NSMenuItem)(obj), C.NSInteger(tag))
	})
	return item
}

func (item *MenuItem) SetSubmenu(menu *Menu) *MenuItem {
	item.WithPointer(func(itemID unsafe.Pointer) {
		menu.WithPointer(func(menuID unsafe.Pointer) {
			C.menuet_ns_menu_item_set_submenu((*C.struct_NSMenuItem)(itemID), (*C.struct_NSMenu)(menuID))
		})
	})
	return item
}

func (item *MenuItem) SetTarget(target unsafe.Pointer) *MenuItem {
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_target((*C.struct_NSMenuItem)(obj), target)
	})
	return item
}

// NewMenuItemCallback invokes callback in a goroutine with an owned sender wrapper.
// The caller must keep the returned target alive while menu items use it.
func NewMenuItemCallback(callback func(*MenuItem)) *core.Callback {
	return core.NewCallback(func(sender core.Sender) {
		item := newMenuItem((*C.struct_NSMenuItem)(unsafe.Pointer(sender.Pointer())))
		go callback(item)
	})
}

func (item *MenuItem) SetAction(action core.Selector) *MenuItem {
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_menu_item_set_action((*C.struct_NSMenuItem)(obj), C.uintptr_t(uintptr(unsafe.Pointer(action.Pointer()))))
	})
	return item
}

// SetCallback sets the target and action. The caller must keep callback alive
// while the menu item uses it.
func (item *MenuItem) SetCallback(callback *core.Callback) *MenuItem {
	callback.WithPointer(func(obj unsafe.Pointer) {
		item.SetTarget(obj).
			SetAction(callback.Selector())
	})
	return item
}

func (item *StatusItem) SetTitle(title string) *StatusItem {
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	item.WithPointer(func(obj unsafe.Pointer) {
		C.menuet_ns_status_item_set_title((*C.struct_NSStatusItem)(obj), value)
	})
	return item
}

func (item *StatusItem) SetImage(image *Image) *StatusItem {
	item.WithPointer(func(itemID unsafe.Pointer) {
		image.WithPointer(func(imageID unsafe.Pointer) {
			C.menuet_ns_status_item_set_image((*C.struct_NSStatusItem)(itemID), (*C.struct_NSImage)(imageID))
		})
	})
	return item
}

func (item *StatusItem) SetMenu(menu *Menu) *StatusItem {
	item.WithPointer(func(itemID unsafe.Pointer) {
		menu.WithPointer(func(menuID unsafe.Pointer) {
			C.menuet_ns_status_item_set_menu((*C.struct_NSStatusItem)(itemID), (*C.struct_NSMenu)(menuID))
		})
	})
	return item
}
