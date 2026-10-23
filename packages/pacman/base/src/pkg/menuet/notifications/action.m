#import "action.h"

struct UNNotificationAction *menuet_un_action_new(const char *identifier, const char *title, UNNotificationActionOptions options) {
	@autoreleasepool {
		UNNotificationAction *result = [UNNotificationAction
			actionWithIdentifier:[NSString stringWithUTF8String:identifier]
			title:[NSString stringWithUTF8String:title]
			options:options];
		return (__bridge_retained struct UNNotificationAction *)result;
	}
}

struct UNNotificationAction *menuet_un_text_input_action_new(const char *identifier, const char *title, UNNotificationActionOptions options, const char *button, const char *placeholder) {
	@autoreleasepool {
		UNNotificationAction *result = [UNTextInputNotificationAction
			actionWithIdentifier:[NSString stringWithUTF8String:identifier]
			title:[NSString stringWithUTF8String:title]
			options:options
			textInputButtonTitle:[NSString stringWithUTF8String:button]
			textInputPlaceholder:[NSString stringWithUTF8String:placeholder]];
		return (__bridge_retained struct UNNotificationAction *)result;
	}
}
