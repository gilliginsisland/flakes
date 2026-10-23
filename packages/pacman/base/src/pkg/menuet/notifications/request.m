#import "request.h"

struct UNNotificationRequest *menuet_un_request_new(const char *identifier, struct UNMutableNotificationContent *content) {
	@autoreleasepool {
		UNNotificationRequest *result = [UNNotificationRequest
			requestWithIdentifier:[NSString stringWithUTF8String:identifier]
			content:(__bridge UNMutableNotificationContent *)content
			trigger:nil];
		return (__bridge_retained struct UNNotificationRequest *)result;
	}
}
