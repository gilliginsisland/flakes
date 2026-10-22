#import "app.h"
#import "../core/core.h"

struct NSApplication *menuet_ns_application_shared(void) {
	__block NSApplication *result;
	menuet_on_main(^{
		result = [NSApplication sharedApplication];
	});
	return (__bridge_retained struct NSApplication *)result;
}
