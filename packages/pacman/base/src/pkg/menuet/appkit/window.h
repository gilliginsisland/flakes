#import <Cocoa/Cocoa.h>

struct NSWindow;

struct NSWindow *menuet_ns_window_new(NSRect contentRect, NSWindowStyleMask styleMask, NSBackingStoreType backing, bool defer);
void menuet_ns_window_set_title(struct NSWindow *window, const char *title);
void menuet_ns_window_set_content_view(struct NSWindow *window, void *view);
void menuet_ns_window_center(struct NSWindow *window);
void menuet_ns_window_make_key_and_order_front(struct NSWindow *window);
void menuet_ns_window_close(struct NSWindow *window);
