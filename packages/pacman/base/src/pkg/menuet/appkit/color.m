#import "color.h"

struct NSColor *menuet_ns_color_clear(void) {
	@autoreleasepool {
		return (__bridge_retained struct NSColor *)NSColor.clearColor;
	}
}
