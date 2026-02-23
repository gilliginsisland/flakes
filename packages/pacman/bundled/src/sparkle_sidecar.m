#import <Cocoa/Cocoa.h>
#import <Sparkle/Sparkle.h>
// #import <objc/runtime.h>

// 1. Define the protocol in your plugin code
@protocol ActionRegistering
- (void)registerAction:(NSString *)action withBlock:(void (^)(id))block;
@end

static SPUStandardUpdaterController* _updateController;

__attribute__((constructor))
static void setup_sparkle_listener() {
	dispatch_async(dispatch_get_main_queue(), ^{
		_updateController = [[SPUStandardUpdaterController alloc] initWithUpdaterDelegate:nil userDriverDelegate:nil];

		// Get the app delegate dynamically
		id delegate = [NSApplication sharedApplication].delegate;
		if (!delegate) {
			return;
		}

		// Dynamically check if the delegate responds to the registration method
		if (![delegate respondsToSelector:NSSelectorFromString(@"registerAction:withBlock:")]) {
			return;
		}
		id<ActionRegistering> registerer = (id<ActionRegistering>)delegate;

		[registerer registerAction:@"sparkle_check_updates" withBlock:^(id data) {
			[_updateController checkForUpdates:nil];
		}];
	});
}
