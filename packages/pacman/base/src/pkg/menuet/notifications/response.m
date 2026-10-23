#import "response.h"
#include <string.h>

char *menuet_un_default_action_identifier(void) {
	@autoreleasepool {
		return strdup(UNNotificationDefaultActionIdentifier.UTF8String);
	}
}

char *menuet_un_dismiss_action_identifier(void) {
	@autoreleasepool {
		return strdup(UNNotificationDismissActionIdentifier.UTF8String);
	}
}

char *menuet_un_response_notification_identifier(struct UNNotificationResponse *response) {
	@autoreleasepool {
		return strdup(((__bridge UNNotificationResponse *)response).notification.request.identifier.UTF8String);
	}
}

char *menuet_un_response_action_identifier(struct UNNotificationResponse *response) {
	@autoreleasepool {
		return strdup(((__bridge UNNotificationResponse *)response).actionIdentifier.UTF8String);
	}
}

char *menuet_un_response_text(struct UNNotificationResponse *response) {
	@autoreleasepool {
		UNNotificationResponse *value = (__bridge UNNotificationResponse *)response;
		if ([value isKindOfClass:UNTextInputNotificationResponse.class]) {
			return strdup(((UNTextInputNotificationResponse *)value).userText.UTF8String);
		}
		return strdup("");
	}
}
