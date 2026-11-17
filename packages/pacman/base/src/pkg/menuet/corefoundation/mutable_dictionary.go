package corefoundation

/*
#include "native.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type MutableDictionaryRef = C.CFMutableDictionaryRef

// MutableDictionary owns a CFMutableDictionaryRef and exposes Dictionary's
// read operations on the same native object.
type MutableDictionary struct{ Dictionary }

func OwnMutableDictionary(ref MutableDictionaryRef) *MutableDictionary {
	if ref == 0 {
		return nil
	}
	value := &MutableDictionary{Dictionary{ref: C.CFDictionaryRef(ref)}}
	runtime.AddCleanup(value, func(ref C.CFMutableDictionaryRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return value
}

func NewMutableDictionary() *MutableDictionary {
	return OwnMutableDictionary(C.CFDictionaryCreateMutable(0, 0, &C.kCFTypeDictionaryKeyCallBacks, &C.kCFTypeDictionaryValueCallBacks))
}

func (value *MutableDictionary) SetString(key StringRef, text string) {
	defer runtime.KeepAlive(value)
	ref := NewString(text)
	defer C.CFRelease(C.CFTypeRef(ref))
	C.menuet_cf_dictionary_set_value(C.CFMutableDictionaryRef(value.ref), key, C.CFTypeRef(ref))
}

func (value *MutableDictionary) SetBool(key StringRef, enabled bool) {
	defer runtime.KeepAlive(value)
	var n C.int
	if enabled {
		n = 1
	}
	ref := C.CFNumberCreate(0, C.kCFNumberIntType, unsafe.Pointer(&n))
	defer C.CFRelease(C.CFTypeRef(ref))
	C.menuet_cf_dictionary_set_value(C.CFMutableDictionaryRef(value.ref), key, C.CFTypeRef(ref))
}

func (value *MutableDictionary) Remove(key StringRef) {
	defer runtime.KeepAlive(value)
	C.menuet_cf_dictionary_remove_value(C.CFMutableDictionaryRef(value.ref), key)
}

// SetValue retains a borrowed native CF reference in the dictionary.
func (value *MutableDictionary) SetValue(key StringRef, ref TypeRef) {
	defer runtime.KeepAlive(value)
	C.menuet_cf_dictionary_set_value(C.CFMutableDictionaryRef(value.ref), key, C.CFTypeRef(ref))
}

func (value *MutableDictionary) SetInt(key StringRef, number int) {
	defer runtime.KeepAlive(value)
	n := C.int64_t(number)
	ref := C.CFNumberCreate(0, C.kCFNumberSInt64Type, unsafe.Pointer(&n))
	defer C.CFRelease(C.CFTypeRef(ref))
	C.menuet_cf_dictionary_set_value(C.CFMutableDictionaryRef(value.ref), key, C.CFTypeRef(ref))
}

func (value *MutableDictionary) SetData(key StringRef, data []byte) {
	defer runtime.KeepAlive(value)
	ref := C.CFDataCreate(0, (*C.UInt8)(unsafe.Pointer(unsafe.SliceData(data))), C.CFIndex(len(data)))
	defer C.CFRelease(C.CFTypeRef(ref))
	C.menuet_cf_dictionary_set_value(C.CFMutableDictionaryRef(value.ref), key, C.CFTypeRef(ref))
}
