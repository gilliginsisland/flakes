package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

#import "const.h"

*/
import "C"
import "unsafe"

func fromNSString(nsstring *C.NSString) string {
	cstr := C.CFStringToUTF8((C.CFStringRef)(unsafe.Pointer(nsstring)))
	return C.GoString(cstr)
}
