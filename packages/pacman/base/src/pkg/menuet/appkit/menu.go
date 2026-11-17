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

type MenuItemRef = *C.struct_NSMenuItem
type StatusItemRef = *C.struct_NSStatusItem
type ViewRef = *C.struct_NSView

type MenuItem struct {
	obj *C.struct_NSMenuItem
}

type StatusItem struct {
	obj *C.struct_NSStatusItem
}

func NewMenu() *Menu {
	return newMenu(C.menuet_ns_menu_new())
}

func newMenu(obj *C.struct_NSMenu) *Menu {
	if obj == nil {
		return nil
	}
	menu := &Menu{obj: obj}
	runtime.AddCleanup(menu, core.Release, core.ObjectRef(menu.obj))
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
	runtime.AddCleanup(item, core.Release, core.ObjectRef(item.obj))
	return item
}

// MenuItemFromPointer takes ownership of a retained Objective-C menu item.
func MenuItemFromPointer(obj MenuItemRef) *MenuItem {
	return newMenuItem(obj)
}

func NewStatusItem() *StatusItem {
	item := &StatusItem{obj: C.menuet_ns_status_item_new()}
	runtime.AddCleanup(item, core.Release, core.ObjectRef(item.obj))
	return item
}

// withPointer permits nil menus when clearing a submenu or status-item menu.
func (menu *Menu) withPointer(fn func(*C.struct_NSMenu)) {
	defer runtime.KeepAlive(menu)
	var obj *C.struct_NSMenu
	if menu != nil {
		obj = menu.obj
	}
	fn(obj)
}

// WithPointer calls fn synchronously while keeping the status item alive.
// A nil status item passes a nil pointer to fn.
func (item *StatusItem) WithPointer(fn func(StatusItemRef)) {
	defer runtime.KeepAlive(item)
	var obj StatusItemRef
	if item != nil {
		obj = item.obj
	}
	fn(obj)
}

func (menu *Menu) Insert(item *MenuItem, index int) {
	defer runtime.KeepAlive(menu)
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_insert(menu.obj, item.obj, C.int(index))
}

func (menu *Menu) Append(items ...*MenuItem) *Menu {
	defer runtime.KeepAlive(menu)
	defer runtime.KeepAlive(items)
	for _, item := range items {
		C.menuet_ns_menu_append(menu.obj, item.obj)
	}
	return menu
}

func (menu *Menu) Remove(item *MenuItem) {
	defer runtime.KeepAlive(menu)
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_remove(menu.obj, item.obj)
}

func (menu *Menu) CancelTracking() {
	defer runtime.KeepAlive(menu)
	C.menuet_ns_menu_cancel_tracking(menu.obj)
}

func (menu *Menu) Items() []*MenuItem {
	defer runtime.KeepAlive(menu)
	values := C.menuet_ns_menu_items(menu.obj)
	defer C.free(unsafe.Pointer(values.items))
	items := make([]*MenuItem, values.count)
	for index, value := range unsafe.Slice(values.items, values.count) {
		items[index] = newMenuItem(value)
	}
	return items
}

func (item *MenuItem) SetTitle(title string) *MenuItem {
	defer runtime.KeepAlive(item)
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_menu_item_set_title(item.obj, value)
	return item
}

func (item *MenuItem) SetSubtitle(subtitle string) *MenuItem {
	defer runtime.KeepAlive(item)
	value := C.CString(subtitle)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_menu_item_set_subtitle(item.obj, value)
	return item
}

type FontWeight float64

var FontWeightSemibold = FontWeight(C.NSFontWeightSemibold)

// SetTitleFont styles the current title using AppKit's attributed-title support.
func (item *MenuItem) SetTitleFont(size float64, weight FontWeight) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_title_font(item.obj, C.double(size), C.double(weight))
	return item
}

func (item *MenuItem) SetBadge(badge string) *MenuItem {
	defer runtime.KeepAlive(item)
	value := C.CString(badge)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_menu_item_set_badge(item.obj, value)
	return item
}

func (item *MenuItem) SetImage(image *Image) *MenuItem {
	defer runtime.KeepAlive(item)
	image.withPointer(func(obj *C.struct_NSImage) {
		C.menuet_ns_menu_item_set_image(item.obj, obj)
	})
	return item
}

type ControlStateValue int

const (
	ControlStateValueOff   ControlStateValue = C.NSControlStateValueOff
	ControlStateValueOn    ControlStateValue = C.NSControlStateValueOn
	ControlStateValueMixed ControlStateValue = C.NSControlStateValueMixed
)

func (item *MenuItem) SetToolTip(text string) *MenuItem {
	defer runtime.KeepAlive(item)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_menu_item_set_tool_tip(item.obj, value)
	return item
}

func (item *MenuItem) SetHidden(hidden bool) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_hidden(item.obj, C.bool(hidden))
	return item
}

// SetView assigns a custom NSView retained by the menu item.
// view must be an Objective-C view pointer or nil, not a Go pointer.
func (item *MenuItem) SetView(view ViewRef) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_view(item.obj, view)
	return item
}

// PopUpMenu queues presentation of the item's menu below status. It skips the
// presentation if the item is hidden before the main run loop handles it.
func (item *MenuItem) PopUpMenu(status *StatusItem) {
	defer runtime.KeepAlive(item)
	defer runtime.KeepAlive(status)
	C.menuet_ns_menu_item_pop_up_menu(item.obj, status.obj)
}

// RepresentedObject returns an owned Objective-C reference. The caller must
// release it or transfer it to a wrapper with cleanup.
func (item *MenuItem) RepresentedObject() core.ObjectRef {
	defer runtime.KeepAlive(item)
	return core.ObjectRef(C.menuet_ns_menu_item_represented_object(item.obj))
}

// SetRepresentedObject sets the Objective-C object retained by the menu item.
// object must be an Objective-C object pointer or nil, not a Go pointer.
func (item *MenuItem) SetRepresentedObject(object core.ObjectRef) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_represented_object(item.obj, (*C.struct_NSObject)(object))
	return item
}

func (item *MenuItem) Tag() int {
	defer runtime.KeepAlive(item)
	return int(C.menuet_ns_menu_item_tag(item.obj))
}

func (item *MenuItem) SetTag(tag int) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_tag(item.obj, C.NSInteger(tag))
	return item
}

// Menu returns an owned wrapper for the containing menu, or nil if unattached.
func (item *MenuItem) Menu() *Menu {
	defer runtime.KeepAlive(item)
	return newMenu(C.menuet_ns_menu_item_menu(item.obj))
}

// Submenu returns an owned wrapper, or nil if the item has no submenu.
func (item *MenuItem) Submenu() *Menu {
	defer runtime.KeepAlive(item)
	return newMenu(C.menuet_ns_menu_item_submenu(item.obj))
}

func (item *MenuItem) SetSubmenu(menu *Menu) *MenuItem {
	defer runtime.KeepAlive(item)
	menu.withPointer(func(obj *C.struct_NSMenu) {
		C.menuet_ns_menu_item_set_submenu(item.obj, obj)
	})
	return item
}

func (item *MenuItem) SetTarget(target core.ObjectRef) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_target(item.obj, (*C.struct_NSObject)(target))
	return item
}

// NewMenuItemCallback invokes callback in a goroutine with an owned menu item sender.
// The caller must keep the returned target alive while menu items use it.
func NewMenuItemCallback(callback func(*MenuItem)) *core.Callback {
	return core.NewCallback(func(sender core.Sender) {
		item := newMenuItem((*C.struct_NSMenuItem)(sender.Pointer()))
		go callback(item)
	})
}

func (item *MenuItem) SetAction(action core.Selector) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_action(item.obj, (*C.struct_objc_selector)(action.Pointer()))
	return item
}

// SetCallback sets the target and action, or clears both for a nil callback.
// The caller must keep callback alive while the menu item uses it.
func (item *MenuItem) SetCallback(callback *core.Callback) *MenuItem {
	var action core.Selector
	if callback != nil {
		action = callback.Selector()
	}
	callback.WithPointer(func(obj core.CallbackRef) {
		item.SetTarget(core.ObjectRef(obj)).
			SetAction(action)
	})
	return item
}

func (item *StatusItem) SetTitle(title string) *StatusItem {
	defer runtime.KeepAlive(item)
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_status_item_set_title(item.obj, value)
	return item
}

func (item *StatusItem) SetImage(image *Image) *StatusItem {
	defer runtime.KeepAlive(item)
	image.withPointer(func(obj *C.struct_NSImage) {
		C.menuet_ns_status_item_set_image(item.obj, obj)
	})
	return item
}

func (item *StatusItem) SetMenu(menu *Menu) *StatusItem {
	defer runtime.KeepAlive(item)
	menu.withPointer(func(obj *C.struct_NSMenu) {
		C.menuet_ns_status_item_set_menu(item.obj, obj)
	})
	return item
}

func (menu *Menu) SetAutoenablesItems(enabled bool) *Menu {
	defer runtime.KeepAlive(menu)
	C.menuet_ns_menu_set_autoenables_items(menu.obj, C.bool(enabled))
	return menu
}
func (item *MenuItem) SetEnabled(enabled bool) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_enabled(item.obj, C.bool(enabled))
	return item
}

func (item *MenuItem) Enabled() bool {
	defer runtime.KeepAlive(item)
	return bool(C.menuet_ns_menu_item_enabled(item.obj))
}

func (item *MenuItem) State() ControlStateValue {
	defer runtime.KeepAlive(item)
	return ControlStateValue(C.menuet_ns_menu_item_state(item.obj))
}

func (item *MenuItem) SetState(state ControlStateValue) *MenuItem {
	defer runtime.KeepAlive(item)
	C.menuet_ns_menu_item_set_state(item.obj, C.NSInteger(state))
	return item
}
