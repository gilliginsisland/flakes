#include <Foundation/Foundation.h>
#import <stdlib.h>
#import <string.h>

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

#import "notification.h"

void go_notification_response_received(NotificationResponse *resp);

Notification* make_notification() {
	return (Notification*)malloc(sizeof(Notification));
}

void destroy_notification(Notification* notification) {
	if (!notification) {
		return;
	}
	free(notification->categoryIdentifier);
	free(notification->identifier);
	free(notification->title);
	free(notification->subtitle);
	free(notification->body);
	free(notification);
}

NotificationAction* make_notification_action_node() {
	NotificationAction* node = (NotificationAction*)malloc(sizeof(NotificationAction));
	*node = (NotificationAction){
		.inputType = NotificationInputTypeNone,
	};
	return node;
}

NotificationActionText* make_notification_action_text_node() {
	NotificationActionText* node = (NotificationActionText*)malloc(sizeof(NotificationActionText));
	*node = (NotificationActionText){
		.action = (NotificationAction){
			.inputType = NotificationInputTypeText,
		},
	};
	return node;
}

void destroy_notification_action_nodes(NotificationAction* node) {
	while (node) {
		NotificationAction* next = node->next;
		free(node->identifier);
		free(node->title);
		if (node->inputType == NotificationInputTypeText) {
			NotificationActionText* textNode = (NotificationActionText*)node;
			free(textNode->buttonTitle);
			free(textNode->placeholder);
		}
		free(node);
		node = next;
	}
}

NotificationCategory* make_notification_category_node() {
	NotificationCategory* category = (NotificationCategory*)malloc(sizeof(NotificationCategory));
	*category = (NotificationCategory){};
	return category;
}

void destroy_notification_category_nodes(NotificationCategory* category) {
	while (category) {
		NotificationCategory* next = category->next;
		free(category->identifier);
		destroy_notification_action_nodes(category->actions);
		free(category);
		category = next;
	}
}

NotificationResponse* make_notification_response() {
	return (NotificationResponse*)malloc(sizeof(NotificationResponse));
}

void destroy_notification_response(NotificationResponse* response) {
	if (!response) {
		return;
	}
	free(response->notificationIdentifier);
	free(response->actionIdentifier);
	free(response->text);
	free(response);
}

NSArray<UNNotificationAction *>* toUNNotificationActions(NotificationAction* actionNode) {
	NSMutableArray *actions = [NSMutableArray<UNNotificationAction *> array];
	while(actionNode) {
		@autoreleasepool {
			UNNotificationAction *action;
			NSString *identifier = [NSString stringWithUTF8String:actionNode->identifier];
			NSString *title = [NSString stringWithUTF8String:actionNode->title];
			if (actionNode->inputType == NotificationInputTypeText) {
				NotificationActionText* textNode = (NotificationActionText*)actionNode;
				NSString *buttonTitle =
					textNode->buttonTitle ? [NSString stringWithUTF8String:textNode->buttonTitle] : @"Send";
				NSString *placeholder =
					textNode->placeholder ? [NSString stringWithUTF8String:textNode->placeholder] : title;
				action = [UNTextInputNotificationAction
					actionWithIdentifier:identifier
					title:title
					options:UNNotificationActionOptionNone
					textInputButtonTitle:buttonTitle
					textInputPlaceholder:placeholder
				];
			} else {
				action = [UNNotificationAction
					actionWithIdentifier:identifier
					title:title
					options:UNNotificationActionOptionForeground
				];
			}
			[actions addObject:action];
		}
		actionNode = actionNode->next;
	}
	return actions;
}

void set_notification_categories(NotificationCategory* categoryNode) {
	@autoreleasepool {
		NSMutableSet *categories = [NSMutableSet<UNNotificationCategory *> new];
		while (categoryNode) {
			@autoreleasepool {
				NSString *identifier = [NSString stringWithUTF8String:categoryNode->identifier];
				UNNotificationCategory *notificationCategory = [UNNotificationCategory
					categoryWithIdentifier:identifier
					actions:toUNNotificationActions(categoryNode->actions)
					intentIdentifiers:@[]
					options:categoryNode->options
				];
				[categories addObject:notificationCategory];
			}
			categoryNode = categoryNode->next;
		}
		[NotificationDelegate sharedInstance].categories = categories;
	}
}

void show_notification(Notification* notification) {
	@autoreleasepool {
		UNMutableNotificationContent *content = [UNMutableNotificationContent new];
		NSString *identifier =
			notification->identifier ?
			[NSString stringWithUTF8String:notification->identifier] :
			[[NSUUID UUID] UUIDString];
		if (notification->categoryIdentifier) {
			content.categoryIdentifier = [NSString stringWithUTF8String:notification->categoryIdentifier];
		}
		if (notification->title) {
			content.title = [NSString stringWithUTF8String:notification->title];
		}
		if (notification->subtitle) {
			content.subtitle = [NSString stringWithUTF8String:notification->subtitle];
		}
		if (notification->body) {
			content.body = [NSString stringWithUTF8String:notification->body];
		}
		UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:identifier content:content trigger:nil];
		[[UNUserNotificationCenter currentNotificationCenter] addNotificationRequest:request withCompletionHandler:^(NSError * _Nullable error) {
			if (error) {
				NSLog(@"Error showing notification: %@", error);
			}
		}];
	}
}

@implementation NotificationDelegate

+ (NotificationDelegate *)sharedInstance {
	static NotificationDelegate *_sharedInstance = nil;
	static dispatch_once_t onceToken;
	dispatch_once(&onceToken, ^{
		_sharedInstance = [NotificationDelegate new];
	});
	return _sharedInstance;
}

- (void)register {
	if (!self.categories) {
		return;
	}
	UNUserNotificationCenter* center = [UNUserNotificationCenter currentNotificationCenter];
	center.delegate = self;

	// Combine standard alert and sound with the provisional option
	UNAuthorizationOptions options =
		UNAuthorizationOptionAlert |
		UNAuthorizationOptionSound |
		UNAuthorizationOptionBadge;

	[center requestAuthorizationWithOptions:options completionHandler:^(BOOL granted, NSError * _Nullable error) {
		if (error) {
			NSLog(@"Error requesting authorization: %@", error);
			[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings * _Nonnull settings) {
				NSLog(@"Current Authorization Status: %ld", (long)settings.authorizationStatus);
				// 1 = Denied, 2 = Authorized, 3 = Provisional
			}];
			return;
		}
		if (granted) {
			NSLog(@"Authorization granted (may be provisional).");
		} else {
			NSLog(@"Authorization denied.");
		}
	}];

	[center setNotificationCategories:self.categories];

}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center didReceiveNotificationResponse:(UNNotificationResponse *)response withCompletionHandler:(void (^)(void))completionHandler {
	NotificationResponse *resp = make_notification_response();
	*resp = (NotificationResponse){
		.notificationIdentifier = strdup([response.notification.request.identifier UTF8String]),
		.actionIdentifier = strdup([response.actionIdentifier UTF8String]),
	};
	if ([response isKindOfClass:[UNTextInputNotificationResponse class]]) {
		UNTextInputNotificationResponse *textResponse = (UNTextInputNotificationResponse *)response;
		resp->text = strdup([textResponse.userText UTF8String]);
	}
	go_notification_response_received(resp);
	completionHandler();
}

@end
