#import <AppKit/AppKit.h>

@protocol PACmanPlugin <NSObject>

// Implementations must keep their menu items' action targets alive independently
// of the caller's reference to the plugin instance.
@property (nonatomic, readonly) NSArray<NSMenuItem *> *menuItems;

@end
