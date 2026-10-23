#import <UserNotifications/UserNotifications.h>
#include <stddef.h>

struct UNNotificationCategory;
struct UNNotificationAction;

struct UNNotificationCategory *menuet_un_category_new(const char *identifier, struct UNNotificationAction **actions, size_t count, UNNotificationCategoryOptions options);
