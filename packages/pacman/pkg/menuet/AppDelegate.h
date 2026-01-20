#import <Cocoa/Cocoa.h>

void runApplication();
void terminateApplication();

@interface AppDelegate : NSObject <NSApplicationDelegate>

+ (AppDelegate *)sharedInstance;

@property (nonatomic, strong, readonly) NSStatusItem *statusItem;

@end
