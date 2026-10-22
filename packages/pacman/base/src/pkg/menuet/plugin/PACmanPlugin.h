#import <AppKit/AppKit.h>

@protocol PACmanPlugin <NSObject>

@property (nonatomic, readonly) NSArray<NSMenuItem *> *menuItems;

@end
