#import <Cocoa/Cocoa.h>
#include <Foundation/Foundation.h>

#import "AppDelegate.h"
#import "menu.h"
#import "notification.h"

void goAppWillFinishLaunching();
void goAppDidFinishLaunching();
void goAppWillTerminate();

@implementation AppDelegate

+ (AppDelegate *)sharedInstance {
    static AppDelegate *_sharedInstance = nil;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        _sharedInstance = [AppDelegate new];
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
    dispatch_async(dispatch_get_main_queue(), ^{
        [[NSApplication sharedApplication] terminate:nil];
    });
}
