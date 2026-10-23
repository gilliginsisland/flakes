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
)

type WebView struct {
	obj *C.struct_WKWebView
}

func NewWebView() *WebView {
	view := &WebView{obj: C.menuet_wk_web_view_new()}
	runtime.AddCleanup(view, core.Release, unsafe.Pointer(view.obj))
	return view
}

func (view *WebView) SetAutoresizingMask(mask appkit.AutoresizingMask) *WebView {
	defer runtime.KeepAlive(view)
	C.menuet_wk_web_view_set_autoresizing_mask(view.obj, C.NSAutoresizingMaskOptions(mask))
	return view
}

func (view *WebView) SetUnderPageBackgroundColor(color unsafe.Pointer) *WebView {
	defer runtime.KeepAlive(view)
	C.menuet_wk_web_view_set_under_page_background_color(view.obj, color)
	return view
}

func (view *WebView) LoadHTMLString(html string, baseURL unsafe.Pointer) {
	defer runtime.KeepAlive(view)
	value := C.CString(html)
	defer C.free(unsafe.Pointer(value))
	C.menuet_wk_web_view_load_html_string(view.obj, value, baseURL)
}

// WithPointer calls fn synchronously while keeping the web view alive.
// A nil web view passes a nil pointer to fn.
func (view *WebView) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(view)
	var obj unsafe.Pointer
	if view != nil {
		obj = unsafe.Pointer(view.obj)
	}
	fn(obj)
}
