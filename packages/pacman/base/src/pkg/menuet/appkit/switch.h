#import <Cocoa/Cocoa.h>
#include <stdbool.h>

// The progress overlay is hosted beside the switch in its menu row.
@interface MenuetSwitch : NSSwitch
@property(nonatomic, strong) NSProgressIndicator *spinner;
@end

struct NSSwitch;
struct MenuetCallback;
struct NSMenu;
struct objc_selector;

NSInteger menuet_ns_switch_state(struct NSSwitch *control);
void menuet_ns_switch_set_state(struct NSSwitch *control, NSInteger state);
bool menuet_ns_switch_enabled(struct NSSwitch *control);
void menuet_ns_switch_set_enabled(struct NSSwitch *control, bool enabled);
void menuet_ns_switch_set_spinning(struct NSSwitch *control, bool spinning);
void menuet_ns_switch_set_callback(struct NSSwitch *control, struct MenuetCallback *callback, struct objc_selector *action);
struct NSMenu *menuet_ns_switch_menu(struct NSSwitch *control);
