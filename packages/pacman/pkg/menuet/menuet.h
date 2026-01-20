#import <Cocoa/Cocoa.h>

void setState(const char *jsonString);
void menuChanged();

@interface MenuetMenu : NSMenu <NSMenuDelegate>

@property(nonatomic, copy) NSString *unique;
@property(nonatomic, assign) BOOL root;
@property(nonatomic, assign) BOOL open;

@end
