#import <Cocoa/Cocoa.h>

struct NSAlert;

struct NSAlert *menuet_ns_alert_new(void);
void menuet_ns_alert_set_message_text(struct NSAlert *alert, const char *text);
void menuet_ns_alert_set_informative_text(struct NSAlert *alert, const char *text);
void menuet_ns_alert_add_button_with_title(struct NSAlert *alert, const char *title);
NSModalResponse menuet_ns_alert_run_modal(struct NSAlert *alert);

struct MenuetPasswordCancellation;
struct MenuetPasswordCancellation *menuet_password_cancellation_new(void);
void menuet_password_cancel(struct MenuetPasswordCancellation *cancellation);
char *menuet_credentials_prompt(const char *message, const char *username, bool needs_password, bool editable_username, bool *remember, char **result_username, struct MenuetPasswordCancellation *cancellation);
NSInteger menuet_account_prompt(const char *message, const char **accounts, size_t count, struct MenuetPasswordCancellation *cancellation);
