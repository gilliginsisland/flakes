package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "color.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Color struct {
	obj *C.struct_NSColor
}

func ClearColor() *Color {
	color := &Color{obj: C.menuet_ns_color_clear()}
	runtime.AddCleanup(color, core.Release, unsafe.Pointer(color.obj))
	return color
}

// WithPointer calls fn synchronously while keeping the color alive.
// A nil color passes a nil pointer to fn.
func (color *Color) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(color)
	var obj unsafe.Pointer
	if color != nil {
		obj = unsafe.Pointer(color.obj)
	}
	fn(obj)
}
