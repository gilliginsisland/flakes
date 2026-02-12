#import <Foundation/Foundation.h>

#import "const.h"

// Explicitly take NSString* and return const char*
const char* CFStringToUTF8(CFStringRef cfstr) {
	if (cfstr == NULL) return NULL;

	// Toll-free bridge: cast CFStringRef to NSString* for ARC compliance
	NSString *nsstr = (__bridge NSString *)cfstr;

	char* result = NULL;
	@autoreleasepool {
		const char *utf8 = [nsstr UTF8String];
		if (utf8) {
			result = strdup(utf8);
		}
	}
	return result;
}
