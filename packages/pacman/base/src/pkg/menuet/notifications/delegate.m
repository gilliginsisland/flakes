#import "delegate.h"
#import "../core/core.h"

@interface MenuetNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@property (weak) MenuetCallback *responseCallback;
@end

@implementation MenuetNotificationDelegate

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
	didReceiveNotificationResponse:(UNNotificationResponse *)response
	withCompletionHandler:(void (^)(void))completionHandler {
	// MenuetCallback passes an owned reference into Go. The response wrapper
	// keeps it alive after this delegate method and its completion return.
	[self.responseCallback invoke:response];
	completionHandler();
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
	willPresentNotification:(UNNotification *)notification
	withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
	NSNumber *options = notification.request.content.userInfo[@"presentationOptions"];
	if ([options isKindOfClass:NSNumber.class]) {
		completionHandler((UNNotificationPresentationOptions)options.unsignedIntegerValue);
	} else {
		completionHandler(UNNotificationPresentationOptionBadge |
			UNNotificationPresentationOptionSound |
			UNNotificationPresentationOptionList |
			UNNotificationPresentationOptionBanner);
	}
}

@end

struct MenuetNotificationDelegate *menuet_un_delegate_new(void) {
	@autoreleasepool {
		return (__bridge_retained struct MenuetNotificationDelegate *)[MenuetNotificationDelegate new];
	}
}

void menuet_un_delegate_set_response_callback(struct MenuetNotificationDelegate *delegate, struct MenuetCallback *callback) {
	@autoreleasepool {
		((__bridge MenuetNotificationDelegate *)delegate).responseCallback = (__bridge MenuetCallback *)callback;
	}
}
