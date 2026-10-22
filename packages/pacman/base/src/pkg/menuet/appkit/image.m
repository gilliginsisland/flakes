#import "image.h"
#import "../core/core.h"

struct NSImage *menuet_ns_image_named(const char *name) {
	__block NSImage *result;
	menuet_on_main(^{
		result = [NSImage imageNamed:[NSString stringWithUTF8String:name]];
	});
	return (__bridge_retained struct NSImage *)result;
}
