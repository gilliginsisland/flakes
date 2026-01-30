package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "menuet.h"

*/
import "C"

import (
	"crypto/rand"
	"sync/atomic"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var menuitems syncutil.WeakMap[string, MenuItem]

// MenuItem represents one item in the dropdown
type MenuItem struct {
	Image string // In Resources dir or URL, should have height 16

	Text       string
	FontSize   int // Default: 14
	FontWeight FontWeight

	State bool // shows checkmark when set

	Clicked  func()
	Children func() []Itemer

	unique atomic.Pointer[string]
}

func (i *MenuItem) item() *C.MenuItem {
	var unique string
	if ptr := i.unique.Load(); ptr != nil {
		unique = *ptr
	} else {
		key := menuitems.StoreRandom(i, rand.Text)
		if i.unique.CompareAndSwap(nil, &key) {
			unique = key
		} else {
			menuitems.Delete(key)
			unique = *i.unique.Load()
		}
	}
	item := (*C.MenuItemRegular)(unsafe.Pointer(C.make_menu_item(C.MenuItemTypeRegular)))
	item.item.unique = C.CString(unique)
	*item = C.MenuItemRegular{
		item:       item.item,
		imageName:  C.CString(i.Image),
		text:       C.CString(i.Text),
		fontSize:   C.int(i.FontSize),
		fontWeight: C.float(i.FontWeight),
		state:      C.bool(i.State),
		clickable:  C.bool(i.Clicked != nil),
	}
	if i.Children != nil {
		item.submenu = toMenuItems(i.Children())
	}
	return &item.item
}

type MenuItemSeparator struct{}

func (i *MenuItemSeparator) item() *C.MenuItem {
	return C.make_menu_item(C.MenuItemTypeSeparator)
}

type MenuItemSectionHeader struct {
	Text string
}

func (i *MenuItemSectionHeader) item() *C.MenuItem {
	item := (*C.MenuItemSectionHeader)(unsafe.Pointer(C.make_menu_item(C.MenuItemTypeSectionHeader)))
	*item = C.MenuItemSectionHeader{
		item: item.item,
		text: C.CString(i.Text),
	}
	return &item.item
}

type Itemer interface {
	item() *C.MenuItem
}

func toMenuItems(items []Itemer) *C.MenuItem {
	var node *C.MenuItem
	curr := &node
	for _, item := range items {
		*curr = item.item()
		curr = &(*curr).next
	}
	return node
}
