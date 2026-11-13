#import <Cocoa/Cocoa.h>

struct MenuetInputMenuItem;
struct MenuetCallback;

struct MenuetInputMenuItem *menuet_input_menu_item_new(const char *placeholder);
char *menuet_input_menu_item_string_value(struct MenuetInputMenuItem *item);
void menuet_input_menu_item_set_string_value(struct MenuetInputMenuItem *item, const char *value);
void menuet_input_menu_item_set_submit_callback(struct MenuetInputMenuItem *item, struct MenuetCallback *callback);
