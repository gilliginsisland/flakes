#import <UserNotifications/UserNotifications.h>
#include <stddef.h>

struct UNUserNotificationCenter;
struct UNNotificationCategory;
struct UNNotificationRequest;

struct UNUserNotificationCenter *menuet_un_center_current(void);
void menuet_un_center_set_delegate(struct UNUserNotificationCenter *center, void *delegate);
void menuet_un_center_set_categories(struct UNUserNotificationCenter *center, struct UNNotificationCategory **categories, size_t count);
void menuet_un_center_request_authorization(struct UNUserNotificationCenter *center, UNAuthorizationOptions options);
void menuet_un_center_add_request(struct UNUserNotificationCenter *center, struct UNNotificationRequest *request);
void menuet_un_center_remove_pending_requests(struct UNUserNotificationCenter *center, const char *identifier);
void menuet_un_center_remove_delivered_notifications(struct UNUserNotificationCenter *center, const char *identifier);
void menuet_un_center_remove_all_pending_requests(struct UNUserNotificationCenter *center);
void menuet_un_center_remove_all_delivered_notifications(struct UNUserNotificationCenter *center);
