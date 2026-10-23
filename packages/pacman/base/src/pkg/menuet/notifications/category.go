package notifications

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework UserNotifications

#include <stdlib.h>
#import "category.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type CategoryOptions uint64

const (
	CategoryOptionNone          CategoryOptions = C.UNNotificationCategoryOptionNone
	CategoryOptionCustomDismiss CategoryOptions = C.UNNotificationCategoryOptionCustomDismissAction
)

type Category struct {
	obj *C.struct_UNNotificationCategory
}

type CategoryContent struct {
	Identifier string
	Actions    []*Action
	Options    CategoryOptions
}

func NewCategory(content CategoryContent) *Category {
	id := C.CString(content.Identifier)
	defer C.free(unsafe.Pointer(id))
	// Native construction retains the actions before the temporary pointer
	// array goes away. Keep their Go wrappers alive through that call.
	defer runtime.KeepAlive(content.Actions)
	values := make([]*C.struct_UNNotificationAction, len(content.Actions))
	for index, action := range content.Actions {
		values[index] = action.obj
	}
	category := &Category{obj: C.menuet_un_category_new(id, unsafe.SliceData(values), C.size_t(len(values)), C.UNNotificationCategoryOptions(content.Options))}
	runtime.AddCleanup(category, core.Release, unsafe.Pointer(category.obj))
	return category
}

// WithPointer calls fn synchronously while keeping the category alive.
// A nil category passes a nil pointer to fn.
func (category *Category) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(category)
	var obj unsafe.Pointer
	if category != nil {
		obj = unsafe.Pointer(category.obj)
	}
	fn(obj)
}
