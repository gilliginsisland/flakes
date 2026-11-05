package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "window.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type WindowStyleMask uint64

const (
	WindowStyleMaskTitled    WindowStyleMask = C.NSWindowStyleMaskTitled
	WindowStyleMaskClosable  WindowStyleMask = C.NSWindowStyleMaskClosable
	WindowStyleMaskResizable WindowStyleMask = C.NSWindowStyleMaskResizable
)

type BackingStoreType uint64

const BackingStoreBuffered BackingStoreType = C.NSBackingStoreBuffered

type WindowOptions struct {
	ContentRect Rect
	StyleMask   WindowStyleMask
	Backing     BackingStoreType
	Defer       bool
}

type Window struct {
	obj *C.struct_NSWindow
}

func NewWindow(options WindowOptions) *Window {
	rect := options.ContentRect
	return newWindow(C.menuet_ns_window_new(
		C.NSMakeRect(C.CGFloat(rect.X), C.CGFloat(rect.Y), C.CGFloat(rect.Width), C.CGFloat(rect.Height)),
		C.NSWindowStyleMask(options.StyleMask), C.NSBackingStoreType(options.Backing), C.bool(options.Defer),
	))
}

func newWindow(obj *C.struct_NSWindow) *Window {
	window := &Window{obj: obj}
	runtime.AddCleanup(window, core.Release, unsafe.Pointer(window.obj))
	return window
}

// WindowFromPointer takes ownership of a retained Objective-C window.
func WindowFromPointer(obj unsafe.Pointer) *Window {
	return newWindow((*C.struct_NSWindow)(obj))
}

func (window *Window) SetTitle(title string) *Window {
	defer runtime.KeepAlive(window)
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_window_set_title(window.obj, value)
	return window
}

func (window *Window) SetContentView(view unsafe.Pointer) *Window {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_set_content_view(window.obj, view)
	return window
}

func (window *Window) Center() {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_center(window.obj)
}

func (window *Window) MakeKeyAndOrderFront() {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_make_key_and_order_front(window.obj)
}

func (window *Window) Close() {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_close(window.obj)
}

// WithPointer calls fn synchronously while keeping the window alive.
// A nil window passes a nil pointer to fn.
func (window *Window) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(window)
	var obj unsafe.Pointer
	if window != nil {
		obj = unsafe.Pointer(window.obj)
	}
	fn(obj)
}

func (window *Window) SetRestorable(value bool) *Window {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_set_restorable(window.obj, C.bool(value))
	return window
}

func (window *Window) SetOpaque(value bool) *Window {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_set_opaque(window.obj, C.bool(value))
	return window
}

func (window *Window) SetAlphaValue(value float64) *Window {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_set_alpha_value(window.obj, C.double(value))
	return window
}

func (window *Window) SetIgnoresMouseEvents(value bool) *Window {
	defer runtime.KeepAlive(window)
	C.menuet_ns_window_set_ignores_mouse_events(window.obj, C.bool(value))
	return window
}
