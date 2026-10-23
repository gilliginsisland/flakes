#import <UserNotifications/UserNotifications.h>

struct MenuetNotificationDelegate;
struct MenuetCallback;

struct MenuetNotificationDelegate *menuet_un_delegate_new(void);
void menuet_un_delegate_set_response_callback(struct MenuetNotificationDelegate *delegate, struct MenuetCallback *callback);
