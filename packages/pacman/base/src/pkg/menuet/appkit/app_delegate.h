#import <Cocoa/Cocoa.h>

struct AppDelegate;
struct MenuetCallback;

struct AppDelegate *menuet_app_delegate_new(void);
void menuet_app_delegate_set_will_finish_launching(struct AppDelegate *delegate, struct MenuetCallback *callback);
void menuet_app_delegate_set_did_finish_launching(struct AppDelegate *delegate, struct MenuetCallback *callback);
