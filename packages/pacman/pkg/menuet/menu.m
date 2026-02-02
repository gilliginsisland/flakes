#include <Foundation/Foundation.h>
#include <Foundation/NSObjCRuntime.h>
#include <stdlib.h>

#import <Cocoa/Cocoa.h>

#import "AppDelegate.h"
#import "menu.h"

void goItemClicked(const char *);

MenuItem* make_menu_item(MenuItemType type) {
	MenuItem* item;
	switch (type) {
		case MenuItemTypeSeparator: {
			item = malloc(sizeof(MenuItem));
			break;
		}
		case MenuItemTypeRegular: {
			item = malloc(sizeof(MenuItemRegular));
			break;
		}
		case MenuItemTypeSectionHeader: {
			item = malloc(sizeof(MenuItemSectionHeader));
			break;
		}
	}
	*item = (MenuItem){ .type = type, };
	return item;
}

void destroy_menu_items(MenuItem* item) {
	while (item) {
		MenuItem* next = item->next;
		free(item->unique);
		switch (item->type) {
			case MenuItemTypeRegular: {
				MenuItemRegular* regular = (MenuItemRegular*)item;
				free(regular->text);
				free(regular->imageName);
				destroy_menu_items(regular->submenu);
				break;
			}
			case MenuItemTypeSeparator: {
				break;
			}
			case MenuItemTypeSectionHeader: {
				MenuItemSectionHeader* header = (MenuItemSectionHeader*)item;
				free(header->text);
				break;
			}
		}
		item=next;
	}
}

@implementation MenuetMenu

- (id)init {
	self = [super init];
	if (self) {
		self.delegate = self;
		self.autoenablesItems = false;
	}
	return self;
}

- (void)populate:(MenuItem*)head {
	int i = 0;
	for (i = 0; head; head=head->next, i++) {
		NSMenuItem *item = nil;
		if (i < self.numberOfItems) {
			item = [self itemAtIndex:i];
		}
		if (head->type == MenuItemTypeSeparator) {
			if (!item.separatorItem) {
				[self insertItem:[NSMenuItem separatorItem] atIndex:i];
			}
			continue;
		} else if (head->type == MenuItemTypeSectionHeader) {
			MenuItemSectionHeader* headerNode = (MenuItemSectionHeader*)head;
			NSString* title = headerNode->text ? [NSString stringWithUTF8String:headerNode->text] : @"";
			if (!item.sectionHeader) {
				[self insertItem:[NSMenuItem sectionHeaderWithTitle:title] atIndex:i];
			} else if (![item.title isEqual:title]) {
				item.title = title;
			}
			continue;
		}

		MenuItemRegular* regular = (MenuItemRegular*)head;
		NSString *unique = regular->item.unique ? [NSString stringWithUTF8String:regular->item.unique] : @"";
		NSString *text = regular->text ? [NSString stringWithUTF8String:regular->text] : @"";
		float fontWeight = regular->fontWeight;
		int fontSize = regular->fontSize;
		BOOL state = regular->state;
		BOOL hasChildren = regular->submenu != NULL;
		BOOL clickable = regular->clickable;

		if (!item || item.separatorItem || item.sectionHeader) {
			item = [self insertItemWithTitle:@"" action:nil keyEquivalent:@"" atIndex:i];
		}
		if (fontSize == 0 && fontWeight == 0) {
			item.title = text;
		} else {
			item.attributedTitle = [[NSAttributedString alloc] initWithString:text attributes:@{
				NSFontAttributeName: [NSFont monospacedDigitSystemFontOfSize:fontSize weight:fontWeight],
			}];
		}
		item.target = self;
		if (clickable) {
			item.action = @selector(press:);
			item.representedObject = unique;
		} else {
			item.action = nil;
			item.representedObject = nil;
		}
		item.state = state ? NSControlStateValueOn : NSControlStateValueOff;
		if (hasChildren) {
			if (!item.submenu) {
				item.submenu = [MenuetMenu new];
			}
			MenuetMenu *menu = (MenuetMenu *)item.submenu;
			menu.unique = unique;
			[menu populate:regular->submenu];
		} else if (item.submenu) {
			item.submenu = nil;
		}
		item.enabled = clickable || hasChildren;
		if (regular->imageName) {
			NSString *imageName = [NSString stringWithUTF8String:regular->imageName];
			NSImage *image = [NSImage imageNamed:imageName];
			image.size = NSMakeSize(16, 16);
			item.image = image;
		}
	}
	while (self.numberOfItems > i) {
		[self removeItemAtIndex:self.numberOfItems - 1];
	}
}

- (void)press:(id)sender {
	NSString *callback = [sender representedObject];
	goItemClicked(callback.UTF8String);
}

@end

void set_state(const char *cTitle, const char *cImageName) {
	NSString *title;
	NSString *imageName;
	@autoreleasepool {
		title = [NSString stringWithUTF8String:cTitle];
		imageName = [NSString stringWithUTF8String:cImageName];
	}
	dispatch_async(dispatch_get_main_queue(), ^{
		NSStatusItem *statusItem = [AppDelegate sharedInstance].statusItem;
		NSImage *image = [NSImage imageNamed:imageName];
		image.size = NSMakeSize(20, 20);
		image.template = true;
		statusItem.button.image = image;
		statusItem.button.imagePosition = NSImageLeft;
	});
}

void menu_changed(MenuItem* head) {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSStatusItem *_statusItem = [AppDelegate sharedInstance].statusItem;
		MenuetMenu *menu = (MenuetMenu *)_statusItem.menu;
		[menu populate:head];
		destroy_menu_items(head);
	});
}
