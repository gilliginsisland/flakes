#import <Cocoa/Cocoa.h>
#import <Sparkle/Sparkle.h>
#import "PACmanPlugin.h"

@interface PACmanUpdaterPlugin : NSObject <PACmanPlugin>
@end

@implementation PACmanUpdaterPlugin {
	SPUStandardUpdaterController *_updater;
	NSArray<NSMenuItem *> *_menuItems;
}

- (instancetype)init {
	self = [super init];
	if (self) {
		_updater = [[SPUStandardUpdaterController alloc] initWithUpdaterDelegate:nil userDriverDelegate:nil];
		NSMenuItem *item = [[NSMenuItem alloc]
			initWithTitle:@"Check for Updates..."
			action:@selector(checkForUpdates:)
			keyEquivalent:@""];
		item.target = _updater;
		_menuItems = @[item];
	}
	return self;
}

- (NSArray<NSMenuItem *> *)menuItems {
	return _menuItems;
}

@end
