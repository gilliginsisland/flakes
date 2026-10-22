package core

/*
#include <stdlib.h>
#include "core.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type String struct {
	obj *C.struct_NSString
}

func NewString(value string) *String {
	text := C.CString(value)
	defer C.free(unsafe.Pointer(text))
	return StringFromPointer(unsafe.Pointer(C.menuet_ns_string_new(text)))
}

// StringFromPointer takes ownership of a retained NSString reference.
func StringFromPointer(pointer unsafe.Pointer) *String {
	value := &String{obj: (*C.struct_NSString)(pointer)}
	runtime.AddCleanup(value, Release, pointer)
	return value
}

// WithPointer calls fn synchronously while keeping the string alive.
// A nil string passes a nil pointer to fn.
// The pointer must not be used after fn returns without retaining it separately.
func (s *String) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(s)
	var obj unsafe.Pointer
	if s != nil {
		obj = unsafe.Pointer(s.obj)
	}
	fn(obj)
}

func (s *String) String() string {
	var value *C.char
	s.WithPointer(func(obj unsafe.Pointer) {
		value = C.menuet_ns_string_value((*C.struct_NSString)(obj))
	})
	defer C.free(unsafe.Pointer(value))
	return C.GoString(value)
}
