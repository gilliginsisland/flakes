#import "switch.h"
#import "../core/core.h"

@implementation MenuetSwitch
@end

NSInteger menuet_ns_switch_state(struct NSSwitch *control) {
	__block NSInteger state;
	menuet_on_main(^{ state = ((__bridge NSSwitch *)control).state; });
	return state;
}

void menuet_ns_switch_set_state(struct NSSwitch *control, NSInteger state) {
	menuet_on_main(^{ ((__bridge NSSwitch *)control).state = state; });
}

bool menuet_ns_switch_enabled(struct NSSwitch *control) {
	__block bool enabled;
	menuet_on_main(^{ enabled = ((__bridge NSSwitch *)control).enabled; });
	return enabled;
}

void menuet_ns_switch_set_enabled(struct NSSwitch *control, bool enabled) {
	menuet_on_main(^{ ((__bridge NSSwitch *)control).enabled = enabled; });
}

void menuet_ns_switch_set_spinning(struct NSSwitch *control, bool spinning) {
	menuet_on_main(^{
		NSProgressIndicator *spinner = ((__bridge MenuetSwitch *)control).spinner;
		if (spinning) {
			[spinner startAnimation:nil];
		} else {
			[spinner stopAnimation:nil];
		}
	});
}

void menuet_ns_switch_set_callback(struct NSSwitch *control, struct MenuetCallback *callback, struct objc_selector *action) {
	menuet_on_main(^{
		NSSwitch *value = (__bridge NSSwitch *)control;
		value.target = (__bridge MenuetCallback *)callback;
		value.action = (SEL)action;
	});
}

struct NSMenu *menuet_ns_switch_menu(struct NSSwitch *control) {
	__block NSMenu *menu;
	menuet_on_main(^{ menu = ((__bridge NSSwitch *)control).enclosingMenuItem.menu; });
	return (__bridge_retained struct NSMenu *)menu;
}
