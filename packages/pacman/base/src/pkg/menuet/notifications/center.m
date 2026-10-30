#import "center.h"

struct UNUserNotificationCenter *menuet_un_center_current(void) {
	@autoreleasepool {
		return (__bridge_retained struct UNUserNotificationCenter *)UNUserNotificationCenter.currentNotificationCenter;
	}
}

void menuet_un_center_set_delegate(struct UNUserNotificationCenter *center, void *delegate) {
	@autoreleasepool {
		((__bridge UNUserNotificationCenter *)center).delegate = (__bridge id<UNUserNotificationCenterDelegate>)delegate;
	}
}

void menuet_un_center_set_categories(struct UNUserNotificationCenter *center, struct UNNotificationCategory **categories, size_t count) {
	@autoreleasepool {
		NSMutableSet<UNNotificationCategory *> *values = [NSMutableSet setWithCapacity:count];
		for (size_t index = 0; index < count; index++) {
			[values addObject:(__bridge UNNotificationCategory *)categories[index]];
		}
		[(__bridge UNUserNotificationCenter *)center setNotificationCategories:values];
	}
}

void menuet_un_center_request_authorization(struct UNUserNotificationCenter *center, UNAuthorizationOptions options) {
	@autoreleasepool {
		UNUserNotificationCenter *value = (__bridge UNUserNotificationCenter *)center;
		[value requestAuthorizationWithOptions:options completionHandler:^(BOOL granted, NSError *error) {
			if (error) {
				NSLog(@"Error requesting notification authorization: %@", error);
				[value getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
					NSLog(@"Current Notification Authorization Status: %ld", (long)settings.authorizationStatus);
				}];
			} else if (!granted) {
				NSLog(@"Notification Authorization denied.");
			}
		}];
	}
}

void menuet_un_center_add_request(struct UNUserNotificationCenter *center, struct UNNotificationRequest *request) {
	@autoreleasepool {
		[(__bridge UNUserNotificationCenter *)center
			addNotificationRequest:(__bridge UNNotificationRequest *)request
			withCompletionHandler:^(NSError *error) {
				if (error) {
					NSLog(@"Error showing notification: %@", error);
				}
			}];
	}
}

void menuet_un_center_remove_pending_requests(struct UNUserNotificationCenter *center, const char *identifier) {
	@autoreleasepool {
		[(__bridge UNUserNotificationCenter *)center removePendingNotificationRequestsWithIdentifiers:@[[NSString stringWithUTF8String:identifier]]];
	}
}

void menuet_un_center_remove_delivered_notifications(struct UNUserNotificationCenter *center, const char *identifier) {
	@autoreleasepool {
		[(__bridge UNUserNotificationCenter *)center removeDeliveredNotificationsWithIdentifiers:@[[NSString stringWithUTF8String:identifier]]];
	}
}

void menuet_un_center_remove_all_pending_requests(struct UNUserNotificationCenter *center) {
	@autoreleasepool {
		[(__bridge UNUserNotificationCenter *)center removeAllPendingNotificationRequests];
	}
}

void menuet_un_center_remove_all_delivered_notifications(struct UNUserNotificationCenter *center) {
	@autoreleasepool {
		[(__bridge UNUserNotificationCenter *)center removeAllDeliveredNotifications];
	}
}
