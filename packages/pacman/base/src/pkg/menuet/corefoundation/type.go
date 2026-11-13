// Package corefoundation wraps native Core Foundation types and their ownership.
package corefoundation

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type TypeRef = C.CFTypeRef
type PropertyListRef = C.CFPropertyListRef

// Type owns a generic Core Foundation reference.
type Type struct{ ref C.CFTypeRef }

// PropertyList mirrors Apple's CFPropertyListRef alias of CFTypeRef.
type PropertyList = Type

// OwnType takes ownership of a Create/Copy reference.
func OwnType(ref TypeRef) *Type {
	if ref == 0 {
		return nil
	}
	value := &Type{ref: ref}
	runtime.AddCleanup(value, Release, ref)
	return value
}

// Retain retains a borrowed native CF reference and returns it with the same
// type. A zero reference is returned unchanged.
func Retain[T ~uintptr](ref T) T {
	if ref != 0 {
		C.CFRetain(C.CFTypeRef(ref))
	}
	return ref
}

// Release relinquishes an owned native reference. A zero reference is ignored.
func Release(ref TypeRef) {
	if ref != 0 {
		C.CFRelease(ref)
	}
}

func (value *Type) String() string {
	defer runtime.KeepAlive(value)
	if value == nil || C.CFGetTypeID(value.ref) != C.CFStringGetTypeID() {
		return ""
	}
	return StringFromRef(C.CFStringRef(value.ref))
}

func (value *Type) Bool() bool {
	defer runtime.KeepAlive(value)
	if value == nil {
		return false
	}
	switch C.CFGetTypeID(value.ref) {
	case C.CFBooleanGetTypeID():
		return C.CFBooleanGetValue(C.CFBooleanRef(value.ref)) != 0
	case C.CFNumberGetTypeID():
		var n C.int
		return C.CFNumberGetValue(C.CFNumberRef(value.ref), C.kCFNumberIntType, unsafe.Pointer(&n)) != 0 && n != 0
	}
	return false
}

// WithPointer calls fn synchronously while keeping the value alive.
// A nil value passes a zero reference to fn.
func (value *Type) WithPointer(fn func(TypeRef)) {
	defer runtime.KeepAlive(value)
	var ref TypeRef
	if value != nil {
		ref = value.ref
	}
	fn(ref)
}
