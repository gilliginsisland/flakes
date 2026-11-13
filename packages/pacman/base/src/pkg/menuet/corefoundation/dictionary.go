package corefoundation

/*
#include "native.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type DictionaryRef = C.CFDictionaryRef

// Dictionary owns a CFDictionaryRef. Borrowed dictionaries are retained so the
// wrapper has a lifetime independent of the object it came from.
type Dictionary struct{ ref C.CFDictionaryRef }

// NewDictionary copies native CF key/value references into an owned immutable
// dictionary, retaining each key and value. K and V must be cgo CF reference
// types, represented as uintptr. Keys must be unique according to CF equality.
// The caller must release the returned dictionary after use.
func NewDictionary[K, V ~uintptr](entries map[K]V) DictionaryRef {
	keys := make([]K, 0, len(entries))
	values := make([]V, 0, len(entries))
	for key, value := range entries {
		keys = append(keys, key)
		values = append(values, value)
	}
	return C.CFDictionaryCreate(
		0,
		(*unsafe.Pointer)(unsafe.Pointer(unsafe.SliceData(keys))),
		(*unsafe.Pointer)(unsafe.Pointer(unsafe.SliceData(values))),
		C.CFIndex(len(entries)),
		&C.kCFTypeDictionaryKeyCallBacks,
		&C.kCFTypeDictionaryValueCallBacks,
	)
}

func OwnDictionary(ref DictionaryRef) *Dictionary {
	if ref == 0 {
		return nil
	}
	value := &Dictionary{ref: ref}
	runtime.AddCleanup(value, func(ref C.CFDictionaryRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return value
}

func (value *Dictionary) Get(key StringRef) *Type {
	defer runtime.KeepAlive(value)
	if value == nil {
		return nil
	}
	return OwnType(Retain(C.menuet_cf_dictionary_get_value(value.ref, key)))
}

func (value *Dictionary) MutableCopy() *MutableDictionary {
	defer runtime.KeepAlive(value)
	if value == nil {
		return NewMutableDictionary()
	}
	return OwnMutableDictionary(C.CFDictionaryCreateMutableCopy(0, 0, value.ref))
}

// WithPointer calls fn synchronously while keeping the dictionary alive.
// A nil dictionary passes a zero reference to fn.
func (value *Dictionary) WithPointer(fn func(DictionaryRef)) {
	defer runtime.KeepAlive(value)
	var ref DictionaryRef
	if value != nil {
		ref = value.ref
	}
	fn(ref)
}
