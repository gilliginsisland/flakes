#import <UserNotifications/UserNotifications.h>

#define NotificationInputTypeNone 0
#define NotificationInputTypeText 1

struct NotificationAction {
	struct NotificationAction* next;
	int inputType;
	char* identifier;
	char* title;
};
typedef struct NotificationAction NotificationAction;

struct NotificationActionText {
	NotificationAction action;
	char* buttonTitle;
	char* placeholder;
};
typedef struct NotificationActionText NotificationActionText;

struct NotificationCategory {
	struct NotificationCategory* next;
	char* identifier;
	NotificationAction* actions;
	int options;
};
typedef struct NotificationCategory NotificationCategory;

struct Notification {
	char* categoryIdentifier;
	char* identifier;
	char* title;
	char* subtitle;
	char* body;
};
typedef struct Notification Notification;

struct NotificationResponse {
	char* notificationIdentifier;
	char* actionIdentifier;
	char* text;
};
typedef struct NotificationResponse NotificationResponse;

Notification* make_notification();
void destroy_notification(Notification* notification);

NotificationAction* make_notification_action_node();
NotificationActionText* make_notification_action_text_node();
void destroy_notification_action_nodes(NotificationAction* node);

NotificationCategory* make_notification_category_node();
void destroy_notification_category_nodes(NotificationCategory* category);

NotificationResponse* make_notification_response();
void destroy_notification_response(NotificationResponse* response);

void set_notification_categories(NotificationCategory* category);
void show_notification(Notification* notification);

@interface NotificationDelegate : NSObject <UNUserNotificationCenterDelegate>

@property (nonatomic, strong) NSSet<UNNotificationCategory *> *categories;

+ (NotificationDelegate *)sharedInstance;

- (void)register;

@end
