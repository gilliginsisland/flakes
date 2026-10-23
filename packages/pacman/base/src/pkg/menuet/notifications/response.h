#import <UserNotifications/UserNotifications.h>

struct UNNotificationResponse;

char *menuet_un_default_action_identifier(void);
char *menuet_un_dismiss_action_identifier(void);
char *menuet_un_response_notification_identifier(struct UNNotificationResponse *response);
char *menuet_un_response_action_identifier(struct UNNotificationResponse *response);
char *menuet_un_response_text(struct UNNotificationResponse *response);
