package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "content.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type PresentationOptions uint64

const (
	PresentationOptionNone   PresentationOptions = 0
	PresentationOptionBadge  PresentationOptions = C.UNNotificationPresentationOptionBadge
	PresentationOptionSound  PresentationOptions = C.UNNotificationPresentationOptionSound
	PresentationOptionList   PresentationOptions = C.UNNotificationPresentationOptionList
	PresentationOptionBanner PresentationOptions = C.UNNotificationPresentationOptionBanner
)

type InterruptionLevel int

const (
	InterruptionLevelPassive       InterruptionLevel = C.UNNotificationInterruptionLevelPassive
	InterruptionLevelActive        InterruptionLevel = C.UNNotificationInterruptionLevelActive
	InterruptionLevelTimeSensitive InterruptionLevel = C.UNNotificationInterruptionLevelTimeSensitive
)

type Content struct {
	obj *C.struct_UNMutableNotificationContent
}

func NewContent() *Content {
	content := &Content{obj: C.menuet_un_content_new()}
	runtime.AddCleanup(content, core.Release, unsafe.Pointer(content.obj))
	return content
}

// WithPointer calls fn synchronously while keeping the content alive.
// A nil content passes a nil pointer to fn.
func (content *Content) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(content)
	var obj unsafe.Pointer
	if content != nil {
		obj = unsafe.Pointer(content.obj)
	}
	fn(obj)
}

func (content *Content) SetCategoryIdentifier(text string) *Content {
	defer runtime.KeepAlive(content)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_content_set_category_identifier(content.obj, value)
	return content
}

func (content *Content) SetTitle(text string) *Content {
	defer runtime.KeepAlive(content)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_content_set_title(content.obj, value)
	return content
}

func (content *Content) SetSubtitle(text string) *Content {
	defer runtime.KeepAlive(content)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_content_set_subtitle(content.obj, value)
	return content
}

func (content *Content) SetBody(text string) *Content {
	defer runtime.KeepAlive(content)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_un_content_set_body(content.obj, value)
	return content
}

func (content *Content) SetInterruptionLevel(value InterruptionLevel) *Content {
	defer runtime.KeepAlive(content)
	C.menuet_un_content_set_interruption_level(content.obj, C.UNNotificationInterruptionLevel(value))
	return content
}

func (content *Content) SetPresentationOptions(value PresentationOptions) *Content {
	defer runtime.KeepAlive(content)
	C.menuet_un_content_set_presentation_options(content.obj, C.UNNotificationPresentationOptions(value))
	return content
}
