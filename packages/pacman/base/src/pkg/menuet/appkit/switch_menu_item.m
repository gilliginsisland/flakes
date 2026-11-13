#import "switch_menu_item.h"
#import "switch.h"
#import "../core/core.h"

// Match MacControlCenterUI's vertical padding and platform-specific menu widths.
static const CGFloat MenuetControlCenterVerticalPadding = 4;
static const CGFloat MenuetControlCenterWidth = 310;
static const CGFloat MenuetControlCenterLegacyWidth = 300;

@interface MenuetSwitchMenuItem : NSMenuItem
@property(nonatomic, strong) MenuetSwitch *control;
- (instancetype)initWithTitle:(NSString *)title;
@end

@interface MenuetSwitchView : NSView
@end

@implementation MenuetSwitchView
- (void)viewDidMoveToWindow {
	[super viewDidMoveToWindow];
	// Give the native switch its active appearance without activating the app.
	[self.window becomeKeyWindow];
}
@end

@implementation MenuetSwitchMenuItem
- (void)setEnabled:(BOOL)enabled {
	[super setEnabled:enabled];
	self.control.enabled = enabled;
}

- (instancetype)initWithTitle:(NSString *)text {
	self = [super initWithTitle:text action:NULL keyEquivalent:@""];
	if (self) {
		NSMenu *sizingMenu = [NSMenu new];
		[sizingMenu addItem:[[NSMenuItem alloc] initWithTitle:text action:NULL keyEquivalent:@""]];
		// Measure the runtime's plain menu insets, including any reserved state column.
		sizingMenu.showsStateColumn = NO;
		CGFloat textWidth = ceil([text sizeWithAttributes:@{NSFontAttributeName: sizingMenu.font}].width);
		CGFloat plainWidth = sizingMenu.size.width;
		CGFloat trailingPadding = (plainWidth - textWidth) / 2;
		sizingMenu.showsStateColumn = YES;
		CGFloat leadingPadding = trailingPadding + sizingMenu.size.width - plainWidth;

		MenuetSwitchView *view = [MenuetSwitchView new];
		view.autoresizingMask = NSViewWidthSizable;

		NSTextField *label = [NSTextField labelWithString:text];
		label.font = [NSFontManager.sharedFontManager convertFont:sizingMenu.font toHaveTrait:NSBoldFontMask];
		label.textColor = NSColor.labelColor;
		label.translatesAutoresizingMaskIntoConstraints = NO;
		[view addSubview:label];

		self.control = [MenuetSwitch new];
		NSSwitch *control = self.control;
		control.enabled = self.enabled;
		control.controlSize = NSControlSizeRegular;
		control.translatesAutoresizingMaskIntoConstraints = NO;
		control.accessibilityLabel = text;
		[view addSubview:control];

		self.control.spinner = [NSProgressIndicator new];
		NSProgressIndicator *spinner = self.control.spinner;
		spinner.style = NSProgressIndicatorStyleSpinning;
		spinner.controlSize = NSControlSizeSmall;
		spinner.indeterminate = YES;
		spinner.displayedWhenStopped = NO;
		spinner.translatesAutoresizingMaskIntoConstraints = NO;
		spinner.accessibilityLabel = @"Updating";
		[spinner sizeToFit];
		[view addSubview:spinner];

		// Keep Control Center's header height while aligning with native menu rows.
		[NSLayoutConstraint activateConstraints:@[
			[label.leadingAnchor constraintEqualToAnchor:view.leadingAnchor constant:leadingPadding],
			[control.leadingAnchor constraintGreaterThanOrEqualToSystemSpacingAfterAnchor:label.trailingAnchor multiplier:1],
			[control.trailingAnchor constraintEqualToAnchor:view.trailingAnchor constant:-trailingPadding],
			[control.topAnchor constraintEqualToAnchor:view.topAnchor constant:MenuetControlCenterVerticalPadding],
			[control.bottomAnchor constraintEqualToAnchor:view.bottomAnchor constant:-MenuetControlCenterVerticalPadding],
			[label.centerYAnchor constraintEqualToAnchor:control.centerYAnchor],
			[label.topAnchor constraintGreaterThanOrEqualToAnchor:view.topAnchor constant:MenuetControlCenterVerticalPadding],
			[label.bottomAnchor constraintLessThanOrEqualToAnchor:view.bottomAnchor constant:-MenuetControlCenterVerticalPadding],
			[spinner.centerXAnchor constraintEqualToAnchor:control.centerXAnchor],
			[spinner.centerYAnchor constraintEqualToAnchor:control.centerYAnchor],
			[spinner.widthAnchor constraintEqualToConstant:spinner.frame.size.width],
			[spinner.heightAnchor constraintEqualToConstant:spinner.frame.size.height],
		]];
		CGFloat width;
		if (@available(macOS 26, *)) {
			width = MenuetControlCenterWidth;
		} else {
			width = MenuetControlCenterLegacyWidth;
		}
		[view setFrameSize:NSMakeSize(width, view.fittingSize.height)];
		[view layoutSubtreeIfNeeded];
		self.view = view;
	}
	return self;
}
@end

struct MenuetSwitchMenuItem *menuet_switch_menu_item_new(const char *title) {
	__block MenuetSwitchMenuItem *item;
	menuet_on_main(^{
		item = [[MenuetSwitchMenuItem alloc] initWithTitle:[NSString stringWithUTF8String:title]];
	});
	return (__bridge_retained struct MenuetSwitchMenuItem *)item;
}

struct NSSwitch *menuet_switch_menu_item_switch(struct MenuetSwitchMenuItem *item) {
	__block NSSwitch *control;
	menuet_on_main(^{
		control = ((__bridge MenuetSwitchMenuItem *)item).control;
	});
	return (__bridge_retained struct NSSwitch *)control;
}
