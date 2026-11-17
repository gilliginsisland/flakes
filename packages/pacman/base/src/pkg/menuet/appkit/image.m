#import "image.h"
#import "../core/core.h"

struct NSImage *menuet_ns_image_named(const char *name) {
	__block NSImage *result;
	menuet_on_main(^{
		result = [NSImage imageNamed:[NSString stringWithUTF8String:name]];
	});
	return (__bridge_retained struct NSImage *)result;
}

struct NSImage *menuet_ns_image_system_symbol(const char *name, MenuetSymbolColor color) {
	__block NSImage *result;
	menuet_on_main(^{
		NSImageSymbolConfiguration *configuration = [NSImageSymbolConfiguration
		    configurationWithPointSize:13 weight:NSFontWeightRegular scale:NSImageSymbolScaleMedium];
		NSColor *tint = nil;
		switch (color) {
		case MenuetSymbolColorGreen: tint = NSColor.systemGreenColor; break;
		case MenuetSymbolColorOrange: tint = NSColor.systemOrangeColor; break;
		case MenuetSymbolColorRed: tint = NSColor.systemRedColor; break;
		case MenuetSymbolColorDefault: break;
		}
		if (tint) {
			configuration = [configuration configurationByApplyingConfiguration:
			    [NSImageSymbolConfiguration configurationWithPaletteColors:@[tint]]];
		}
		result = [[NSImage imageWithSystemSymbolName:[NSString stringWithUTF8String:name]
		                    accessibilityDescription:nil] imageWithSymbolConfiguration:configuration];
	});
	return (__bridge_retained struct NSImage *)result;
}
