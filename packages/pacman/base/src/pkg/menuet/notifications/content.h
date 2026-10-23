#import <UserNotifications/UserNotifications.h>

struct UNMutableNotificationContent;

struct UNMutableNotificationContent *menuet_un_content_new(void);
void menuet_un_content_set_category_identifier(struct UNMutableNotificationContent *content, const char *value);
void menuet_un_content_set_title(struct UNMutableNotificationContent *content, const char *value);
void menuet_un_content_set_subtitle(struct UNMutableNotificationContent *content, const char *value);
void menuet_un_content_set_body(struct UNMutableNotificationContent *content, const char *value);
void menuet_un_content_set_interruption_level(struct UNMutableNotificationContent *content, UNNotificationInterruptionLevel value);
void menuet_un_content_set_presentation_options(struct UNMutableNotificationContent *content, UNNotificationPresentationOptions value);
