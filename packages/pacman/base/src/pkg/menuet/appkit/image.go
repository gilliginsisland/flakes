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

// WithPointer calls fn synchronously while keeping the image alive.
// A nil image passes a nil pointer to fn.
func (image *Image) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(image)
	var obj unsafe.Pointer
	if image != nil {
		obj = unsafe.Pointer(image.obj)
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
	runtime.AddCleanup(&image, core.Release, unsafe.Pointer(image.obj))
	return &image
}
