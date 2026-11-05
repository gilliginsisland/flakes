#import <AppKit/AppKit.h>
#import <AuthenticationServices/AuthenticationServices.h>

struct NSWindow;
struct NSWindow *menuet_as_anchor_window_new(NSRect contentRect, NSWindowStyleMask styleMask, NSBackingStoreType backing, bool defer);
