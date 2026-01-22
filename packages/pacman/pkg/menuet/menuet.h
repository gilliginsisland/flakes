#import <Cocoa/Cocoa.h>

void setState(const char *cTitle, const char *cImageName);
void menuChanged();

@interface MenuetMenu : NSMenu <NSMenuDelegate>

@property(nonatomic, copy) NSString *unique;
@property(nonatomic, assign) BOOL root;
@property(nonatomic, assign) BOOL open;

@end
