#import <UserNotifications/UserNotifications.h>

struct UNNotificationRequest;
struct UNMutableNotificationContent;

struct UNNotificationRequest *menuet_un_request_new(const char *identifier, struct UNMutableNotificationContent *content);
