#import <Foundation/Foundation.h>
#import <UserNotifications/UNUserNotificationCenter.h>

typedef enum NotificationInputType {
	NotificationInputTypeNone,
	NotificationInputTypeText,
} NotificationInputType;

typedef struct NotificationAction {
	struct NotificationAction* next;
	NotificationInputType inputType;
	char* identifier;
	char* title;
} NotificationAction;

typedef struct NotificationActionText {
	NotificationAction action;
	char* buttonTitle;
	char* placeholder;
} NotificationActionText;

typedef struct NotificationCategory {
	struct NotificationCategory* next;
	char* identifier;
	NotificationAction* actions;
	int options;
} NotificationCategory;

typedef struct Notification {
	char* categoryIdentifier;
	char* identifier;
	char* title;
	char* subtitle;
	char* body;
} Notification;

typedef struct NotificationResponse {
	char* notificationIdentifier;
	char* actionIdentifier;
	char* text;
} NotificationResponse;

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
