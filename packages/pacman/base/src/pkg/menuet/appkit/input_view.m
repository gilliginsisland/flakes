#import "input_view.h"
#import "EditableNSTextField.h"
#import "../core/core.h"

#include <string.h>

@interface MenuetInputView : NSView
@property(nonatomic, strong) EditableNSTextField *field;
@end

@implementation MenuetInputView

- (void)viewDidMoveToWindow {
	[super viewDidMoveToWindow];
	if (self.window) {
		[NSApp activateIgnoringOtherApps:YES];
		[self.window makeFirstResponder:self.field];
	}
}

@end

struct MenuetInputView *menuet_input_view_new(const char *placeholder) {
	__block MenuetInputView *result;
	menuet_on_main(^{
		result = [MenuetInputView new];
		result.autoresizingMask = NSViewWidthSizable;
		result.field = [EditableNSTextField new];
		result.field.placeholderString = [NSString stringWithUTF8String:placeholder];
		result.field.translatesAutoresizingMaskIntoConstraints = NO;
		[result addSubview:result.field];
		NSLayoutGuide *margins = result.layoutMarginsGuide;
		[NSLayoutConstraint activateConstraints:@[
			[result.field.leadingAnchor constraintEqualToAnchor:margins.leadingAnchor],
			[result.field.trailingAnchor constraintEqualToAnchor:margins.trailingAnchor],
			[result.field.topAnchor constraintEqualToAnchor:margins.topAnchor],
			[result.field.bottomAnchor constraintEqualToAnchor:margins.bottomAnchor],
		]];
		[result setFrameSize:NSMakeSize(330, result.fittingSize.height)];
		[result layoutSubtreeIfNeeded];
	});
	return (__bridge_retained struct MenuetInputView *)result;
}

char *menuet_input_view_string_value(struct MenuetInputView *view) {
	__block char *result;
	menuet_on_main(^{
		result = strdup(((__bridge MenuetInputView *)view).field.stringValue.UTF8String);
	});
	return result;
}

void menuet_input_view_set_string_value(struct MenuetInputView *view, const char *value) {
	menuet_on_main(^{
		((__bridge MenuetInputView *)view).field.stringValue = [NSString stringWithUTF8String:value];
	});
}

void menuet_input_view_set_submit_callback(struct MenuetInputView *view, struct MenuetCallback *callback) {
	menuet_on_main(^{
		EditableNSTextField *value = ((__bridge MenuetInputView *)view).field;
		value.target = (__bridge MenuetCallback *)callback;
		value.action = @selector(invoke:);
	});
}
