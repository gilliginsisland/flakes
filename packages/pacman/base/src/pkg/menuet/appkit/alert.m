#import "alert.h"
#import "../core/core.h"

struct NSAlert *menuet_ns_alert_new(void) {
	__block NSAlert *result;
	menuet_on_main(^{
		result = [NSAlert new];
	});
	return (__bridge_retained struct NSAlert *)result;
}

void menuet_ns_alert_set_message_text(struct NSAlert *alert, const char *text) {
	menuet_on_main(^{
		((__bridge NSAlert *)alert).messageText = [NSString stringWithUTF8String:text];
	});
}

void menuet_ns_alert_set_informative_text(struct NSAlert *alert, const char *text) {
	menuet_on_main(^{
		((__bridge NSAlert *)alert).informativeText = [NSString stringWithUTF8String:text];
	});
}

void menuet_ns_alert_add_button_with_title(struct NSAlert *alert, const char *title) {
	menuet_on_main(^{
		[(__bridge NSAlert *)alert addButtonWithTitle:[NSString stringWithUTF8String:title]];
	});
}

NSModalResponse menuet_ns_alert_run_modal(struct NSAlert *alert) {
	__block NSModalResponse result;
	menuet_on_main(^{
		[NSApp activateIgnoringOtherApps:YES];
		result = [(__bridge NSAlert *)alert runModal];
	});
	return result;
}
