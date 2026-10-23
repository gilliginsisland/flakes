#import <Cocoa/Cocoa.h>

struct NSApplication;

struct NSApplication *menuet_ns_application_shared(void);
void menuet_ns_application_run(struct NSApplication *application);
void menuet_ns_application_terminate(struct NSApplication *application);
void menuet_ns_application_set_delegate(struct NSApplication *application, void *delegate);
void menuet_ns_application_set_activation_policy(struct NSApplication *application, NSApplicationActivationPolicy policy);
void menuet_ns_application_activate_ignoring_other_apps(struct NSApplication *application, bool flag);
