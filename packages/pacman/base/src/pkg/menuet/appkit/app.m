#import "app.h"
#import "../core/core.h"

struct NSApplication *menuet_ns_application_shared(void) {
	__block NSApplication *result;
	menuet_on_main(^{
		result = [NSApplication sharedApplication];
	});
	return (__bridge_retained struct NSApplication *)result;
}

void menuet_ns_application_run(struct NSApplication *application) {
	@autoreleasepool {
		[(__bridge NSApplication *)application run];
	}
}

void menuet_ns_application_terminate(struct NSApplication *application) {
	NSApplication *app = (__bridge NSApplication *)application;
	[[NSRunLoop mainRunLoop] performInModes:@[NSRunLoopCommonModes] block:^{
		[app terminate:nil];
	}];
}

void menuet_ns_application_set_delegate(struct NSApplication *application, void *delegate) {
	menuet_on_main(^{
		((__bridge NSApplication *)application).delegate = (__bridge id<NSApplicationDelegate>)delegate;
	});
}

void menuet_ns_application_set_activation_policy(struct NSApplication *application, NSApplicationActivationPolicy policy) {
	menuet_on_main(^{
		[(__bridge NSApplication *)application setActivationPolicy:policy];
	});
}
