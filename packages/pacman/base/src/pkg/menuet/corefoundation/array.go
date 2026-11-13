package corefoundation

/*
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

import "unsafe"

type ArrayRef = C.CFArrayRef

// NewArray copies native CF references into an owned immutable array, retaining
// each item. T must be a cgo Core Foundation reference type, represented as uintptr.
// The Go slice is only borrowed during this call; its storage is not retained.
// The caller must release the returned array after use.
func NewArray[T ~uintptr](values []T) ArrayRef {
	return C.CFArrayCreate(0, (*unsafe.Pointer)(unsafe.Pointer(unsafe.SliceData(values))), C.CFIndex(len(values)), &C.kCFTypeArrayCallBacks)
}
