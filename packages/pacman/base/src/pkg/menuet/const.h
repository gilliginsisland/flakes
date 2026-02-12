#import <CoreFoundation/CoreFoundation.h>

// Pass the NSString as a CFStringRef to bypass ARC pointer checks
const char* CFStringToUTF8(CFStringRef str);
