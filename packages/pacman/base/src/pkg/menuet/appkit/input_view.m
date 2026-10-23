#import "input_view.h"
#import "EditableNSTextField.h"
#import "../core/core.h"

#include <string.h>

@interface MenuetInputView : NSView

@property (nonatomic, strong) NSTextField *message;
@property (nonatomic, strong) NSTextField *input;
@property (nonatomic, strong) NSButton *submit;
@property (nonatomic, strong) NSButton *cancel;

@end

@implementation MenuetInputView

- (void)viewDidMoveToWindow {
	[super viewDidMoveToWindow];
	if (self.window) {
		// Menu tracking uses its own run-loop mode. Focus after the view has
		// been attached, and skip if the prompt was dismissed in the meantime.
		[[NSRunLoop mainRunLoop] performInModes:@[NSRunLoopCommonModes] block:^{
			if (self.window) {
				[NSApp activateIgnoringOtherApps:YES];
				[self.window makeFirstResponder:self.input];
			}
		}];
	}
}

@end

struct MenuetInputView *menuet_input_view_new(const char *message, const char *placeholder) {
	__block MenuetInputView *result;
	menuet_on_main(^{
		result = [[MenuetInputView alloc] initWithFrame:NSMakeRect(0, 0, 330, 124)];
		result.message = [NSTextField labelWithString:[NSString stringWithUTF8String:message]];
		result.message.frame = NSMakeRect(16, 94, 298, 20);
		result.message.autoresizingMask = NSViewWidthSizable;
		[result addSubview:result.message];

		result.input = [[EditableNSTextField alloc] initWithFrame:NSMakeRect(16, 57, 298, 26)];
		result.input.placeholderString = [NSString stringWithUTF8String:placeholder];
		result.input.autoresizingMask = NSViewWidthSizable;
		[result addSubview:result.input];

		result.submit = [NSButton buttonWithTitle:@"Submit" target:nil action:NULL];
		result.submit.frame = NSMakeRect(218, 12, 96, 30);
		result.submit.autoresizingMask = NSViewMinXMargin;
		[result addSubview:result.submit];
		result.cancel = [NSButton buttonWithTitle:@"Cancel" target:nil action:NULL];
		result.cancel.frame = NSMakeRect(114, 12, 96, 30);
		result.cancel.autoresizingMask = NSViewMinXMargin;
		[result addSubview:result.cancel];
	});
	return (__bridge_retained struct MenuetInputView *)result;
}

char *menuet_input_view_string_value(struct MenuetInputView *view) {
	__block char *result;
	menuet_on_main(^{
		result = strdup(((__bridge MenuetInputView *)view).input.stringValue.UTF8String);
	});
	return result;
}

void menuet_input_view_set_message_text(struct MenuetInputView *view, const char *message) {
	menuet_on_main(^{
		((__bridge MenuetInputView *)view).message.stringValue = [NSString stringWithUTF8String:message];
	});
}

void menuet_input_view_set_string_value(struct MenuetInputView *view, const char *value) {
	menuet_on_main(^{
		((__bridge MenuetInputView *)view).input.stringValue = [NSString stringWithUTF8String:value];
	});
}

void menuet_input_view_set_submit_callback(struct MenuetInputView *view, struct MenuetCallback *callback) {
	menuet_on_main(^{
		MenuetInputView *value = (__bridge MenuetInputView *)view;
		value.input.target = (__bridge MenuetCallback *)callback;
		value.input.action = @selector(invoke:);
		value.submit.target = (__bridge MenuetCallback *)callback;
		value.submit.action = @selector(invoke:);
	});
}

void menuet_input_view_set_cancel_callback(struct MenuetInputView *view, struct MenuetCallback *callback) {
	menuet_on_main(^{
		MenuetInputView *value = (__bridge MenuetInputView *)view;
		value.cancel.target = (__bridge MenuetCallback *)callback;
		value.cancel.action = @selector(invoke:);
	});
}
