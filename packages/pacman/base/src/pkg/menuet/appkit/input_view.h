#import <Cocoa/Cocoa.h>

struct MenuetInputView;
struct MenuetCallback;

struct MenuetInputView *menuet_input_view_new(const char *placeholder);
char *menuet_input_view_string_value(struct MenuetInputView *view);
void menuet_input_view_set_string_value(struct MenuetInputView *view, const char *value);
void menuet_input_view_set_submit_callback(struct MenuetInputView *view, struct MenuetCallback *callback);
