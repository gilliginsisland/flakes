#import <Foundation/Foundation.h>

#import "core.h"
#include <string.h>

void menuet_on_main(void (^block)(void)) {
	void (^work)(void) = ^{
		@autoreleasepool {
			block();
		}
	};
	if (NSThread.isMainThread) {
		work();
	} else {
		dispatch_sync(dispatch_get_main_queue(), work);
	}
}

@interface MenuetCallback : NSObject
@end

@implementation MenuetCallback

- (void)invoke:(NSObject *)sender {
	menuet_core_invoke_callback((uintptr_t)(__bridge void *)self, (__bridge_retained void *)sender);
}

@end

struct MenuetCallback *menuet_callback_new(void) {
	return (__bridge_retained struct MenuetCallback *)[MenuetCallback new];
}

SEL menuet_selector(const char *name) {
	return sel_registerName(name);
}

struct NSString *menuet_ns_string_new(const char *value) {
	return (__bridge_retained struct NSString *)[[NSString alloc] initWithUTF8String:value];
}

char *menuet_ns_string_value(struct NSString *string) {
	return strdup(((__bridge NSString *)string).UTF8String);
}
