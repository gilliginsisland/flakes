#import "menu.h"
#import "../core/core.h"

struct NSMenu *menuet_ns_menu_new(void) {
	__block NSMenu *result;
	menuet_on_main(^{
		result = [NSMenu new];
	});
	return (__bridge_retained struct NSMenu *)result;
}

void menuet_ns_menu_insert(struct NSMenu *menu, struct NSMenuItem *item, int index) {
	menuet_on_main(^{
		[(__bridge NSMenu *)menu insertItem:(__bridge NSMenuItem *)item atIndex:index];
	});
}

void menuet_ns_menu_append(struct NSMenu *menu, struct NSMenuItem *item) {
	menuet_on_main(^{
		[(__bridge NSMenu *)menu addItem:(__bridge NSMenuItem *)item];
	});
}

void menuet_ns_menu_remove(struct NSMenu *menu, struct NSMenuItem *item) {
	menuet_on_main(^{
		[(__bridge NSMenu *)menu removeItem:(__bridge NSMenuItem *)item];
	});
}

menuet_ns_menu_item_array menuet_ns_menu_items(struct NSMenu *menu) {
	__block menuet_ns_menu_item_array result;
	menuet_on_main(^{
		NSArray<NSMenuItem *> *itemArray = ((__bridge NSMenu *)menu).itemArray;
		result.count = itemArray.count;
		result.items = calloc(result.count, sizeof(*result.items));
		for (NSUInteger index = 0; index < result.count; index++) {
			result.items[index] = (__bridge_retained struct NSMenuItem *)itemArray[index];
		}
	});
	return result;
}

struct NSMenuItem *menuet_ns_menu_item_new(void) {
	__block NSMenuItem *result;
	menuet_on_main(^{
		result = [NSMenuItem new];
	});
	return (__bridge_retained struct NSMenuItem *)result;
}

struct NSMenuItem *menuet_ns_menu_item_separator(void) {
	__block NSMenuItem *result;
	menuet_on_main(^{
		result = [NSMenuItem separatorItem];
	});
	return (__bridge_retained struct NSMenuItem *)result;
}

struct NSMenuItem *menuet_ns_menu_item_section_header(const char *title) {
	__block NSMenuItem *result;
	menuet_on_main(^{
		result = [NSMenuItem sectionHeaderWithTitle:[NSString stringWithUTF8String:title]];
	});
	return (__bridge_retained struct NSMenuItem *)result;
}

void menuet_ns_menu_item_set_title(struct NSMenuItem *item, const char *title) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).title = [NSString stringWithUTF8String:title];
	});
}

void menuet_ns_menu_item_set_badge(struct NSMenuItem *item, const char *badge) {
	menuet_on_main(^{
		NSMenuItemBadge *nsbadge = nil;
		if (badge[0]) {
			nsbadge =
				[[NSMenuItemBadge alloc] initWithString:[NSString stringWithUTF8String:badge]];
		}
		((__bridge NSMenuItem *)item).badge = nsbadge;
	});
}

void menuet_ns_menu_item_set_hidden(struct NSMenuItem *item, bool hidden) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).hidden = hidden;
	});
}

void *menuet_ns_menu_item_represented_object(struct NSMenuItem *item) {
	__block id result;
	menuet_on_main(^{
		result = ((__bridge NSMenuItem *)item).representedObject;
	});
	return (__bridge_retained void *)result;
}

void menuet_ns_menu_item_set_represented_object(struct NSMenuItem *item, void *object) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).representedObject = (__bridge id)object;
	});
}

NSInteger menuet_ns_menu_item_tag(struct NSMenuItem *item) {
	__block NSInteger result;
	menuet_on_main(^{
		result = ((__bridge NSMenuItem *)item).tag;
	});
	return result;
}

void menuet_ns_menu_item_set_tag(struct NSMenuItem *item, NSInteger tag) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).tag = tag;
	});
}

void menuet_ns_menu_item_set_submenu(struct NSMenuItem *item, struct NSMenu *menu) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).submenu = (__bridge NSMenu *)menu;
	});
}

void menuet_ns_menu_item_set_target(struct NSMenuItem *item, void *target) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).target = (__bridge id)target;
	});
}

void menuet_ns_menu_item_set_action(struct NSMenuItem *item, uintptr_t action) {
	menuet_on_main(^{
		((__bridge NSMenuItem *)item).action = (SEL)(void *)action;
	});
}

struct NSStatusItem *menuet_ns_status_item_new(void) {
	__block NSStatusItem *result;
	menuet_on_main(^{
		result = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
	});
	return (__bridge_retained struct NSStatusItem *)result;
}

void menuet_ns_status_item_set_title(struct NSStatusItem *item, const char *title) {
	menuet_on_main(^{
		((__bridge NSStatusItem *)item).button.title = [NSString stringWithUTF8String:title];
	});
}

void menuet_ns_status_item_set_image(struct NSStatusItem *item, struct NSImage *image) {
	menuet_on_main(^{
		NSImage *value = (__bridge NSImage *)image;
		value.size = NSMakeSize(20, 20);
		value.template = YES;
		((__bridge NSStatusItem *)item).button.image = value;
	});
}

void menuet_ns_status_item_set_menu(struct NSStatusItem *item, struct NSMenu *menu) {
	menuet_on_main(^{
		((__bridge NSStatusItem *)item).menu = (__bridge NSMenu *)menu;
	});
}
