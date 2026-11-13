#import <Cocoa/Cocoa.h>

struct NSAlert;

struct NSAlert *menuet_ns_alert_new(void);
void menuet_ns_alert_set_message_text(struct NSAlert *alert, const char *text);
void menuet_ns_alert_set_informative_text(struct NSAlert *alert, const char *text);
void menuet_ns_alert_add_button_with_title(struct NSAlert *alert, const char *title);
NSModalResponse menuet_ns_alert_run_modal(struct NSAlert *alert);
