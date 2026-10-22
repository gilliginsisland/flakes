#import "app_delegate.h"
#import "../core/core.h"

@interface AppDelegate : NSObject <NSApplicationDelegate>

@property (nonatomic, weak) MenuetCallback *willFinishLaunching;
@property (nonatomic, weak) MenuetCallback *didFinishLaunching;

@end

@implementation AppDelegate

- (void)applicationWillFinishLaunching:(NSNotification *)notification {
	[self.willFinishLaunching invoke:notification];
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	[self.didFinishLaunching invoke:notification];
}

@end

struct AppDelegate *menuet_app_delegate_new(void) {
	__block AppDelegate *result;
	menuet_on_main(^{
		result = [AppDelegate new];
	});
	return (__bridge_retained struct AppDelegate *)result;
}

void menuet_app_delegate_set_will_finish_launching(struct AppDelegate *delegate, struct MenuetCallback *callback) {
	menuet_on_main(^{
		((__bridge AppDelegate *)delegate).willFinishLaunching = (__bridge MenuetCallback *)callback;
	});
}

void menuet_app_delegate_set_did_finish_launching(struct AppDelegate *delegate, struct MenuetCallback *callback) {
	menuet_on_main(^{
		((__bridge AppDelegate *)delegate).didFinishLaunching = (__bridge MenuetCallback *)callback;
	});
}
