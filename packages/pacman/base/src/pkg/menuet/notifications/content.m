#import "content.h"

struct UNMutableNotificationContent *menuet_un_content_new(void) {
	@autoreleasepool {
		return (__bridge_retained struct UNMutableNotificationContent *)[UNMutableNotificationContent new];
	}
}

void menuet_un_content_set_category_identifier(struct UNMutableNotificationContent *content, const char *value) {
	@autoreleasepool {
		((__bridge UNMutableNotificationContent *)content).categoryIdentifier = [NSString stringWithUTF8String:value];
	}
}

void menuet_un_content_set_title(struct UNMutableNotificationContent *content, const char *value) {
	@autoreleasepool {
		((__bridge UNMutableNotificationContent *)content).title = [NSString stringWithUTF8String:value];
	}
}

void menuet_un_content_set_subtitle(struct UNMutableNotificationContent *content, const char *value) {
	@autoreleasepool {
		((__bridge UNMutableNotificationContent *)content).subtitle = [NSString stringWithUTF8String:value];
	}
}

void menuet_un_content_set_body(struct UNMutableNotificationContent *content, const char *value) {
	@autoreleasepool {
		((__bridge UNMutableNotificationContent *)content).body = [NSString stringWithUTF8String:value];
	}
}

void menuet_un_content_set_interruption_level(struct UNMutableNotificationContent *content, UNNotificationInterruptionLevel value) {
	@autoreleasepool {
		if (@available(macOS 12.0, *)) {
			((__bridge UNMutableNotificationContent *)content).interruptionLevel = value;
		}
	}
}

void menuet_un_content_set_presentation_options(struct UNMutableNotificationContent *content, UNNotificationPresentationOptions value) {
	@autoreleasepool {
		((__bridge UNMutableNotificationContent *)content).userInfo = @{ @"presentationOptions": @(value) };
	}
}
