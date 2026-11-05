#import "anchor_window.h"
#import "../core/core.h"

@interface MenuetAnchorWindow : NSWindow <ASWebAuthenticationPresentationContextProviding>
@end

@implementation MenuetAnchorWindow
- (ASPresentationAnchor)presentationAnchorForWebAuthenticationSession:(ASWebAuthenticationSession *)session {
	return self;
}
@end

struct NSWindow *menuet_as_anchor_window_new(NSRect contentRect, NSWindowStyleMask styleMask, NSBackingStoreType backing, bool defer) {
	__block MenuetAnchorWindow *result;
	menuet_on_main(^{
		result = [[MenuetAnchorWindow alloc] initWithContentRect:contentRect styleMask:styleMask backing:backing defer:defer];
		// The Go Window wrapper owns this reference and releases it through cleanup.
		result.releasedWhenClosed = NO;
	});
	return (__bridge_retained struct NSWindow *)result;
}
