// Package corefoundation wraps native Core Foundation types and their ownership.
package corefoundation

/*
#cgo LDFLAGS: -framework CoreFoundation
#include "native.h"
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

// Data copies a CFData value into Go-owned memory.
func (value *Type) Data() []byte {
	defer runtime.KeepAlive(value)
	if value == nil || C.CFGetTypeID(value.ref) != C.CFDataGetTypeID() {
		return nil
	}
	ref := C.CFDataRef(value.ref)
	return C.GoBytes(unsafe.Pointer(C.CFDataGetBytePtr(ref)), C.int(C.CFDataGetLength(ref)))
}

// Dictionary retains a dictionary value independently of its generic owner.
func (value *Type) Dictionary() *Dictionary {
	defer runtime.KeepAlive(value)
	if value == nil || C.CFGetTypeID(value.ref) != C.CFDictionaryGetTypeID() {
		return nil
	}
	return OwnDictionary(Retain(DictionaryRef(value.ref)))
}

// Array retains each array element independently of the array's lifetime.
func (value *Type) Array() []*Type {
	defer runtime.KeepAlive(value)
	if value == nil || C.CFGetTypeID(value.ref) != C.CFArrayGetTypeID() {
		return nil
	}
	ref := C.CFArrayRef(value.ref)
	values := make([]*Type, int(C.CFArrayGetCount(ref)))
	for i := range values {
		values[i] = OwnType(Retain(C.menuet_cf_array_get_value(ref, C.CFIndex(i))))
	}
	return values
}
