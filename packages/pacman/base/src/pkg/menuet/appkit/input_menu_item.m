#import "input_menu_item.h"
#import "../core/core.h"
#import "EditableNSTextField.h"

#include <string.h>

@interface MenuetInputMenuItem : NSMenuItem
@property(nonatomic, strong) EditableNSTextField *field;
- (instancetype)initWithPlaceholder:(NSString *)placeholder;
@end

@interface MenuetInputView : NSView
@property(nonatomic, weak) EditableNSTextField *field;
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

@implementation MenuetInputMenuItem
- (instancetype)initWithPlaceholder:(NSString *)placeholder {
	self = [super initWithTitle:@"" action:NULL keyEquivalent:@""];
	if (self) {
		MenuetInputView *view = [MenuetInputView new];
		view.autoresizingMask = NSViewWidthSizable;
		self.field = [EditableNSTextField new];
		view.field = self.field;
		self.field.placeholderString = placeholder;
		self.field.translatesAutoresizingMaskIntoConstraints = NO;
		[view addSubview:self.field];
		NSLayoutGuide *margins = view.layoutMarginsGuide;
		[NSLayoutConstraint activateConstraints:@[
			[self.field.leadingAnchor constraintEqualToAnchor:margins.leadingAnchor],
			[self.field.trailingAnchor constraintEqualToAnchor:margins.trailingAnchor],
			[self.field.topAnchor constraintEqualToAnchor:margins.topAnchor],
			[self.field.bottomAnchor constraintEqualToAnchor:margins.bottomAnchor],
		]];
		[view setFrameSize:NSMakeSize(330, view.fittingSize.height)];
		[view layoutSubtreeIfNeeded];
		self.view = view;
	}
	return self;
}
@end

struct MenuetInputMenuItem *menuet_input_menu_item_new(const char *placeholder) {
	__block MenuetInputMenuItem *item;
	menuet_on_main(^{
		item = [[MenuetInputMenuItem alloc] initWithPlaceholder:[NSString stringWithUTF8String:placeholder]];
	});
	return (__bridge_retained struct MenuetInputMenuItem *)item;
}

char *menuet_input_menu_item_string_value(struct MenuetInputMenuItem *item) {
	__block char *result;
	menuet_on_main(^{
		result = strdup(((__bridge MenuetInputMenuItem *)item).field.stringValue.UTF8String);
	});
	return result;
}

void menuet_input_menu_item_set_string_value(struct MenuetInputMenuItem *item, const char *value) {
	menuet_on_main(^{
		((__bridge MenuetInputMenuItem *)item).field.stringValue = [NSString stringWithUTF8String:value];
	});
}

void menuet_input_menu_item_set_submit_callback(struct MenuetInputMenuItem *item, struct MenuetCallback *callback) {
	menuet_on_main(^{
		EditableNSTextField *value = ((__bridge MenuetInputMenuItem *)item).field;
		value.target = (__bridge MenuetCallback *)callback;
		value.action = @selector(invoke:);
	});
}
