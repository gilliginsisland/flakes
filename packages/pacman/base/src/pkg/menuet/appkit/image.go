package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "image.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Image struct{ obj *C.struct_NSImage }

type SymbolColor int

const (
	SymbolColorDefault SymbolColor = C.MenuetSymbolColorDefault
	SymbolColorGreen   SymbolColor = C.MenuetSymbolColorGreen
	SymbolColorOrange  SymbolColor = C.MenuetSymbolColorOrange
	SymbolColorRed     SymbolColor = C.MenuetSymbolColorRed
)

// ImageSystemSymbol creates a menu-sized SF Symbol with an optional semantic tint.
func ImageSystemSymbol(name string, color SymbolColor) *Image {
	value := C.CString(name)
	defer C.free(unsafe.Pointer(value))
	image := &Image{obj: C.menuet_ns_image_system_symbol(value, C.MenuetSymbolColor(color))}
	if image.obj == nil {
		return nil
	}
	runtime.AddCleanup(image, core.Release, core.ObjectRef(image.obj))
	return image
}

// withPointer permits nil images when clearing menu-item and status-item images.
func (image *Image) withPointer(fn func(*C.struct_NSImage)) {
	defer runtime.KeepAlive(image)
	var obj *C.struct_NSImage
	if image != nil {
		obj = image.obj
	}
	fn(obj)
}

func ImageNamed(name string) *Image {
	value := C.CString(name)
	defer C.free(unsafe.Pointer(value))
	image := Image{obj: C.menuet_ns_image_named(value)}
	if image.obj == nil {
		return nil
	}
	runtime.AddCleanup(&image, core.Release, core.ObjectRef(image.obj))
	return &image
}
