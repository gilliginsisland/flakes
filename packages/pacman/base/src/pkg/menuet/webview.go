package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#include <stdlib.h>

#import "webview.h"

*/
import "C"

import "unsafe"

func WebView(html string) {
	cstr := C.CString(html)
	defer C.free(unsafe.Pointer(cstr))
	C.openWebView(cstr)
}
