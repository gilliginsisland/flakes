#import <Cocoa/Cocoa.h>

#import "AppDelegate.h"
#import "notification.h"

void goAppWillFinishLaunching();
void goAppDidFinishLaunching();
void goAppWillTerminate();

@implementation AppDelegate

NSMutableDictionary<NSString *, void (^)(id)> *_actions;

+ (AppDelegate *)sharedInstance {
	static AppDelegate *_sharedInstance = nil;
	static dispatch_once_t onceToken;
	dispatch_once(&onceToken, ^{
		_sharedInstance = [AppDelegate new];
		_actions = [NSMutableDictionary new]; // Initialize the app actions dictionary
	});
	return _sharedInstance;
}

- (void)applicationWillFinishLaunching:(NSNotification *)notification {
	[[NotificationDelegate sharedInstance] register];
	goAppWillFinishLaunching();
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	goAppDidFinishLaunching();
}

- (void)applicationWillTerminate:(NSNotification *)notification {
	goAppWillTerminate();
}

- (void)registerAction:(NSString *)action withBlock:(void (^)(id))block {
	_actions[action] = block;
}

- (void)invokeAction:(NSString *)action withData:(id)data {
	void (^block)(id) = _actions[action];
	if (action) {
		block(data);
	}
}

- (BOOL)hasAction:(NSString *)action {
	return _actions[action] != nil;
}

@end

void runApplication() {
	@autoreleasepool {
		NSApplication *a = [NSApplication sharedApplication];
		[a setDelegate:[AppDelegate sharedInstance]];
		[a setActivationPolicy:NSApplicationActivationPolicyAccessory];
		[a run];
	}
}

void terminateApplication() {
	[[NSRunLoop mainRunLoop] performInModes:@[NSRunLoopCommonModes] block: ^{
		[[NSApplication sharedApplication] terminate:nil];
	}];
}

void invoke_app_action(const char *action) {
	@autoreleasepool {
		[[AppDelegate sharedInstance] invokeAction:[NSString stringWithUTF8String:action] withData:nil];
	}
}

bool has_app_action(const char *action) {
	@autoreleasepool {
		return [[AppDelegate sharedInstance] hasAction:[NSString stringWithUTF8String:action]] ? true : false;
	}
}
