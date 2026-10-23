#import "webview.h"
#import "../core/core.h"

struct WKWebView *menuet_wk_web_view_new(void) {
	__block WKWebView *result;
	menuet_on_main(^{
		result = [WKWebView new];
	});
	return (__bridge_retained struct WKWebView *)result;
}

void menuet_wk_web_view_set_autoresizing_mask(struct WKWebView *view, NSAutoresizingMaskOptions mask) {
	menuet_on_main(^{
		((__bridge WKWebView *)view).autoresizingMask = mask;
	});
}

void menuet_wk_web_view_set_under_page_background_color(struct WKWebView *view, void *color) {
	menuet_on_main(^{
		((__bridge WKWebView *)view).underPageBackgroundColor = (__bridge NSColor *)color;
	});
}

void menuet_wk_web_view_load_html_string(struct WKWebView *view, const char *html, void *baseURL) {
	menuet_on_main(^{
		// Convert and hand the HTML to WebKit before Go frees the C string.
		[(__bridge WKWebView *)view loadHTMLString:[NSString stringWithUTF8String:html] baseURL:(__bridge NSURL *)baseURL];
	});
}

void menuet_wk_web_view_load_request(struct WKWebView *view, struct NSURLRequest *request) {
	menuet_on_main(^{
		[(__bridge WKWebView *)view loadRequest:(__bridge NSURLRequest *)request];
	});
}
