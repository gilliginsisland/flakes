#import "url.h"

struct NSURL *menuet_ns_url_with_string(const char *value) {
	@autoreleasepool {
		return (__bridge_retained struct NSURL *)[NSURL URLWithString:[NSString stringWithUTF8String:value]];
	}
}
