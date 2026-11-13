package webkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework WebKit

#include <stdlib.h>
#import "webview.h"
*/
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
	"github.com/gilliginsisland/pacman/pkg/menuet/foundation"
)

type WebViewRef = *C.struct_WKWebView

type WebView struct {
	obj *C.struct_WKWebView
}

func NewWebView() *WebView {
	view := &WebView{obj: C.menuet_wk_web_view_new()}
	runtime.AddCleanup(view, core.Release, core.ObjectRef(view.obj))
	return view
}

// WithPointer borrows the native web view synchronously, keeping its owner alive.
// A nil web view passes nil to fn.
func (view *WebView) WithPointer(fn func(WebViewRef)) {
	defer runtime.KeepAlive(view)
	var ref WebViewRef
	if view != nil {
		ref = view.obj
	}
	fn(ref)
}

func (view *WebView) SetAutoresizingMask(mask appkit.AutoresizingMask) *WebView {
	defer runtime.KeepAlive(view)
	C.menuet_wk_web_view_set_autoresizing_mask(view.obj, C.NSAutoresizingMaskOptions(mask))
	return view
}

func (view *WebView) SetUnderPageBackgroundColor(color *appkit.Color) *WebView {
	defer runtime.KeepAlive(view)
	color.WithPointer(func(ref appkit.ColorRef) {
		C.menuet_wk_web_view_set_under_page_background_color(view.obj, (*C.struct_NSColor)(ref))
	})
	return view
}

func (view *WebView) LoadHTMLString(html string, baseURL *foundation.URL) {
	defer runtime.KeepAlive(view)
	value := C.CString(html)
	defer C.free(unsafe.Pointer(value))
	baseURL.WithPointer(func(ref foundation.URLRef) {
		C.menuet_wk_web_view_load_html_string(view.obj, value, (*C.struct_NSURL)(ref))
	})
}

func (view *WebView) LoadRequest(request *foundation.URLRequest) {
	defer runtime.KeepAlive(view)
	request.WithPointer(func(ref foundation.URLRequestRef) {
		C.menuet_wk_web_view_load_request(view.obj, (*C.struct_NSURLRequest)(ref))
	})
}
