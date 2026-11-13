package corefoundation

/*
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
*/
import "C"

import "unsafe"

type StringRef = C.CFStringRef

// NewString creates an owned native string. The caller must release it after use.
func NewString(text string) StringRef {
	bytes := C.CString(text)
	defer C.free(unsafe.Pointer(bytes))
	return C.CFStringCreateWithCString(0, bytes, C.kCFStringEncodingUTF8)
}

// StringFromRef copies a borrowed CF string. The caller keeps its owner alive.
func StringFromRef(ref StringRef) string {
	if ref == 0 {
		return ""
	}
	size := C.CFStringGetMaximumSizeForEncoding(C.CFStringGetLength(ref), C.kCFStringEncodingUTF8) + 1
	buf := make([]byte, int(size))
	if C.CFStringGetCString(ref, (*C.char)(unsafe.Pointer(&buf[0])), size, C.kCFStringEncodingUTF8) == 0 {
		return ""
	}
	return C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
}
