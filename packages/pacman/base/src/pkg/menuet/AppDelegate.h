#import <Cocoa/Cocoa.h>

void runApplication();
void terminateApplication();

@interface AppDelegate : NSObject <NSApplicationDelegate>

+ (AppDelegate *)sharedInstance;
- (void)registerAction:(NSString *)action withBlock:(void (^)(id))block;
- (void)invokeAction:(NSString *)action withData:(id)data;
- (BOOL)hasAction:(NSString *)action;

@end
