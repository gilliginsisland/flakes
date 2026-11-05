package authenticationservices

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AuthenticationServices -framework Cocoa
#import "anchor_window.h"
*/
import "C"

import (
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
)

// AnchorWindow is a window that returns itself as the session's
// presentation anchor. Its embedded Window owns the native reference and cleanup.
type AnchorWindow struct {
	*appkit.Window
}

func NewAnchorWindow(options appkit.WindowOptions) *AnchorWindow {
	rect := options.ContentRect
	obj := C.menuet_as_anchor_window_new(
		C.NSMakeRect(C.CGFloat(rect.X), C.CGFloat(rect.Y), C.CGFloat(rect.Width), C.CGFloat(rect.Height)),
		C.NSWindowStyleMask(options.StyleMask), C.NSBackingStoreType(options.Backing), C.bool(options.Defer),
	)
	return &AnchorWindow{Window: appkit.WindowFromPointer(unsafe.Pointer(obj))}
}
