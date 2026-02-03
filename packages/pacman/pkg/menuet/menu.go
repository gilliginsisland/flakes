package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "menu.h"

*/
import "C"

import (
	"crypto/rand"
	"log/slog"
	"sync/atomic"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var (
	menuitems   syncutil.WeakMap[string, MenuItem]
	statusItems syncutil.WeakMap[string, StatusItem]
)

// StatusItem represents a status bar item with a title, image, and submenu
type StatusItem struct {
	Title   string
	Image   string // In Resources dir or URL, should have height 22
	Submenu Itemer
	unique  atomic.Pointer[string]
}

func (i *StatusItem) item() *C.StatusItem {
	var unique string
	if ptr := i.unique.Load(); ptr != nil {
		unique = *ptr
	} else {
		key := statusItems.StoreRandom(i, rand.Text)
		if i.unique.CompareAndSwap(nil, &key) {
			unique = key
		} else {
			statusItems.Delete(key)
			unique = *i.unique.Load()
		}
	}
	item := (*C.StatusItem)(unsafe.Pointer(C.make_status_item()))
	*item = C.StatusItem{
		unique:    C.CString(unique),
		title:     C.CString(i.Title),
		imageName: C.CString(i.Image),
	}
	if i.Submenu != nil {
		item.submenu = i.Submenu.item()
	}
	return item
}

// UpdateStatusItem adds or updates a status item in the status bar
func (a *Application) UpdateStatusItem(item *StatusItem) {
	C.update_status_item(item.item())
}

// RemoveStatusItem removes a status item from the status bar
func (a *Application) RemoveStatusItem(item *StatusItem) {
	unique := item.unique.Load()
	if unique == nil {
		return
	}
	cstr := C.CString(*unique)
	defer C.free(unsafe.Pointer(cstr))
	defer statusItems.Delete(*unique)
	C.remove_status_item(cstr)
}

// MenuItem represents one item in the dropdown
type MenuItem struct {
	Image string // In Resources dir or URL, should have height 16

	Text       string
	FontSize   int // Default: 14
	FontWeight FontWeight

	State    bool // shows checkmark when set
	Subtitle string

	Clicked func()
	Submenu Itemer
	Badge   string

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
		subtitle:   C.CString(i.Subtitle),
		badge:      C.CString(i.Badge),
	}
	if i.Submenu != nil {
		item.submenu = i.Submenu.item()
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

type Section struct {
	Title   string
	Content Itemer
}

func (s *Section) item() *C.MenuItem {
	return toMenuItems([]Itemer{
		&MenuItemSectionHeader{
			Text: s.Title,
		},
		s.Content,
	})
}

type Sections []Itemer

func (ss Sections) item() *C.MenuItem {
	var children []Itemer
	n := len(ss)
	if n == 0 {
		return nil
	}
	children = make([]Itemer, n*2)

	for i, m := range ss {
		children[i*2] = m
		children[i*2+1] = &MenuItemSeparator{}
	}

	return toMenuItems(children[0 : n*2-1])
}

type MenuItems []Itemer

func (s MenuItems) item() *C.MenuItem {
	return toMenuItems(s)
}

type DynamicItem func() Itemer

func (f DynamicItem) item() *C.MenuItem {
	return f().item()
}

type DynamicItems func() []Itemer

func (f DynamicItems) item() *C.MenuItem {
	return toMenuItems(f())
}

//export goItemClicked
func goItemClicked(cUnique *C.char) {
	unique := C.GoString(cUnique)
	go func() {
		item := menuitems.Load(unique)
		if item == nil {
			slog.Debug("Item not found for click", slog.String("unique", unique))
			return
		}
		if item.Clicked == nil {
			slog.Debug("Item has no click handler", slog.String("unique", unique))
			return
		}
		item.Clicked()
	}()
}

type Itemer interface {
	item() *C.MenuItem
}

func toMenuItems(items []Itemer) *C.MenuItem {
	var head *C.MenuItem
	tail := &head
	for _, item := range items {
		*tail = item.item()
		for (*tail) != nil {
			tail = &(*tail).next
		}
	}
	return head
}
