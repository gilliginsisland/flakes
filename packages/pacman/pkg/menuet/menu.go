package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "menu.h"

*/
import "C"

import (
	"crypto/rand"
	"log"
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

	Clicked func()
	Submenu Itemer

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
