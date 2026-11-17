#import <Cocoa/Cocoa.h>

struct NSImage;

struct NSImage *menuet_ns_image_named(const char *name);

typedef NS_ENUM(NSInteger, MenuetSymbolColor) {
	MenuetSymbolColorDefault,
	MenuetSymbolColorGreen,
	MenuetSymbolColorOrange,
	MenuetSymbolColorRed,
};

struct NSImage *menuet_ns_image_system_symbol(const char *name, MenuetSymbolColor color);
