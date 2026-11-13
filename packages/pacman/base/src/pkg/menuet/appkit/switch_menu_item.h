#import <Cocoa/Cocoa.h>

struct MenuetSwitchMenuItem;

struct MenuetSwitchMenuItem *menuet_switch_menu_item_new(const char *title);
struct NSSwitch *menuet_switch_menu_item_switch(struct MenuetSwitchMenuItem *item);
