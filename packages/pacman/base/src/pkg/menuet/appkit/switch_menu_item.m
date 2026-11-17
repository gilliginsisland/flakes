#import "switch_menu_item.h"
#import "switch.h"
#import "../core/core.h"

// Match MacControlCenterUI's vertical padding.
static const CGFloat MenuetControlCenterVerticalPadding = 4;

@interface MenuetSwitchMenuItem : NSMenuItem
@property(nonatomic, strong) MenuetSwitch *control;
@property(nonatomic, strong) NSTextField *titleLabel;
@property(nonatomic, strong) NSTextField *subtitleLabel;
- (instancetype)initWithTitle:(NSString *)title;
- (void)sizeToFit;
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
- (void)setTitle:(NSString *)title {
	[super setTitle:title];
	self.titleLabel.stringValue = title;
	self.control.accessibilityLabel = title;
	[self sizeToFit];
}

- (void)setSubtitle:(NSString *)subtitle {
	[super setSubtitle:subtitle];
	self.subtitleLabel.stringValue = subtitle ?: @"";
	self.subtitleLabel.hidden = subtitle.length == 0;
	[self sizeToFit];
}

- (void)sizeToFit {
	NSView *view = self.view;
	if (view) {
		[view setFrameSize:view.fittingSize];
		[view layoutSubtreeIfNeeded];
	}
}

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
		self.titleLabel = label;
		label.font = [NSFontManager.sharedFontManager convertFont:sizingMenu.font toHaveTrait:NSBoldFontMask];
		label.textColor = NSColor.labelColor;
		label.translatesAutoresizingMaskIntoConstraints = NO;

		NSTextField *subtitle = [NSTextField labelWithString:self.subtitle ?: @""];
		self.subtitleLabel = subtitle;
		subtitle.font = [NSFont systemFontOfSize:NSFont.smallSystemFontSize];
		subtitle.textColor = NSColor.secondaryLabelColor;
		subtitle.lineBreakMode = NSLineBreakByTruncatingTail;
		subtitle.hidden = subtitle.stringValue.length == 0;
		subtitle.translatesAutoresizingMaskIntoConstraints = NO;
		[subtitle setContentCompressionResistancePriority:NSLayoutPriorityDefaultLow
		                                  forOrientation:NSLayoutConstraintOrientationHorizontal];

		NSStackView *labels = [NSStackView stackViewWithViews:@[label, subtitle]];
		labels.orientation = NSUserInterfaceLayoutOrientationHorizontal;
		labels.alignment = NSLayoutAttributeFirstBaseline;
		labels.spacing = 8;
		labels.detachesHiddenViews = YES;
		labels.translatesAutoresizingMaskIntoConstraints = NO;
		[view addSubview:labels];

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
			[labels.leadingAnchor constraintEqualToAnchor:view.leadingAnchor constant:leadingPadding],
			[control.leadingAnchor constraintGreaterThanOrEqualToSystemSpacingAfterAnchor:labels.trailingAnchor multiplier:1],
			[control.trailingAnchor constraintEqualToAnchor:view.trailingAnchor constant:-trailingPadding],
			[control.topAnchor constraintEqualToAnchor:view.topAnchor constant:MenuetControlCenterVerticalPadding],
			[control.bottomAnchor constraintEqualToAnchor:view.bottomAnchor constant:-MenuetControlCenterVerticalPadding],
			[labels.centerYAnchor constraintEqualToAnchor:control.centerYAnchor],
			[labels.topAnchor constraintGreaterThanOrEqualToAnchor:view.topAnchor constant:MenuetControlCenterVerticalPadding],
			[labels.bottomAnchor constraintLessThanOrEqualToAnchor:view.bottomAnchor constant:-MenuetControlCenterVerticalPadding],
			[spinner.centerXAnchor constraintEqualToAnchor:control.centerXAnchor],
			[spinner.centerYAnchor constraintEqualToAnchor:control.centerYAnchor],
			[spinner.widthAnchor constraintEqualToConstant:spinner.frame.size.width],
			[spinner.heightAnchor constraintEqualToConstant:spinner.frame.size.height],
		]];
		self.view = view;
		[self sizeToFit];
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
