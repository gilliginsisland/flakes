#import "window.h"
#import "../core/core.h"

struct NSWindow *menuet_ns_window_new(NSRect contentRect, NSWindowStyleMask styleMask, NSBackingStoreType backing, bool defer) {
	__block NSWindow *result;
	menuet_on_main(^{
		result = [[NSWindow alloc] initWithContentRect:contentRect styleMask:styleMask backing:backing defer:defer];
		// The Go wrapper owns this reference and releases it through cleanup.
		result.releasedWhenClosed = NO;
	});
	return (__bridge_retained struct NSWindow *)result;
}

void menuet_ns_window_set_title(struct NSWindow *window, const char *title) {
	menuet_on_main(^{
		((__bridge NSWindow *)window).title = [NSString stringWithUTF8String:title];
	});
}

void menuet_ns_window_set_content_view(struct NSWindow *window, void *view) {
	menuet_on_main(^{
		((__bridge NSWindow *)window).contentView = (__bridge NSView *)view;
	});
}

void menuet_ns_window_center(struct NSWindow *window) {
	menuet_on_main(^{
		[(__bridge NSWindow *)window center];
	});
}

void menuet_ns_window_make_key_and_order_front(struct NSWindow *window) {
	NSWindow *value = (__bridge NSWindow *)window;
	menuet_on_main(^{
		[value makeKeyAndOrderFront:nil];
	});
}

void menuet_ns_window_close(struct NSWindow *window) {
	menuet_on_main(^{
		[(__bridge NSWindow *)window close];
	});
}
