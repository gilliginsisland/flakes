package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "color.h"
*/
import "C"

import (
	"runtime"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type ColorRef = *C.struct_NSColor

type Color struct {
	obj *C.struct_NSColor
}

func ClearColor() *Color {
	color := &Color{obj: C.menuet_ns_color_clear()}
	runtime.AddCleanup(color, core.Release, core.ObjectRef(color.obj))
	return color
}

// WithPointer borrows the native color synchronously, keeping its owner alive.
// A nil color passes nil to fn.
func (color *Color) WithPointer(fn func(ColorRef)) {
	defer runtime.KeepAlive(color)
	var ref ColorRef
	if color != nil {
		ref = color.obj
	}
	fn(ref)
}
