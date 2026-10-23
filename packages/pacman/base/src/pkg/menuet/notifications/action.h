#import <UserNotifications/UserNotifications.h>

struct UNNotificationAction;

struct UNNotificationAction *menuet_un_action_new(const char *identifier, const char *title, UNNotificationActionOptions options);
struct UNNotificationAction *menuet_un_text_input_action_new(const char *identifier, const char *title, UNNotificationActionOptions options, const char *button, const char *placeholder);
