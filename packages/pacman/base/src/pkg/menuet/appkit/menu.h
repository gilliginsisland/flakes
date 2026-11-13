#import <Cocoa/Cocoa.h>

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

struct NSView;
struct NSMenu;
struct NSMenuItem;
struct NSStatusItem;
struct NSImage;
struct objc_selector;
struct NSObject;

void menuet_ns_menu_item_set_title_font(struct NSMenuItem *item, double size, double weight);

struct NSMenu *menuet_ns_menu_item_menu(struct NSMenuItem *item);

// The caller owns each retained item and must free the raw array separately.
typedef struct menuet_ns_menu_item_array {
	struct NSMenuItem **items;
	size_t count;
} menuet_ns_menu_item_array;

struct NSMenu *menuet_ns_menu_new(void);

void menuet_ns_menu_insert(struct NSMenu *menu, struct NSMenuItem *item, int index);
void menuet_ns_menu_append(struct NSMenu *menu, struct NSMenuItem *item);
void menuet_ns_menu_remove(struct NSMenu *menu, struct NSMenuItem *item);
menuet_ns_menu_item_array menuet_ns_menu_items(struct NSMenu *menu);
void menuet_ns_menu_cancel_tracking(struct NSMenu *menu);

struct NSMenuItem *menuet_ns_menu_item_new(void);
struct NSMenuItem *menuet_ns_menu_item_separator(void);
struct NSMenuItem *menuet_ns_menu_item_section_header(const char *title);

void menuet_ns_menu_item_set_title(struct NSMenuItem *item, const char *title);
void menuet_ns_menu_item_set_image(struct NSMenuItem *item, struct NSImage *image);
void menuet_ns_menu_item_set_badge(struct NSMenuItem *item, const char *badge);
void menuet_ns_menu_item_set_tool_tip(struct NSMenuItem *item, const char *text);
void menuet_ns_menu_item_set_hidden(struct NSMenuItem *item, bool hidden);
void menuet_ns_menu_item_set_view(struct NSMenuItem *item, struct NSView *view);
void menuet_ns_menu_item_pop_up_menu(struct NSMenuItem *item, struct NSStatusItem *status);
struct NSObject *menuet_ns_menu_item_represented_object(struct NSMenuItem *item);
void menuet_ns_menu_item_set_represented_object(struct NSMenuItem *item, struct NSObject *object);
NSInteger menuet_ns_menu_item_tag(struct NSMenuItem *item);
void menuet_ns_menu_item_set_tag(struct NSMenuItem *item, NSInteger tag);
struct NSMenu *menuet_ns_menu_item_submenu(struct NSMenuItem *item);
void menuet_ns_menu_item_set_submenu(struct NSMenuItem *item, struct NSMenu *menu);
void menuet_ns_menu_item_set_target(struct NSMenuItem *item, struct NSObject *target);
void menuet_ns_menu_item_set_action(struct NSMenuItem *item, struct objc_selector *action);

struct NSStatusItem *menuet_ns_status_item_new(void);

void menuet_ns_status_item_set_title(struct NSStatusItem *item, const char *title);
void menuet_ns_status_item_set_image(struct NSStatusItem *item, struct NSImage *image);
void menuet_ns_status_item_set_menu(struct NSStatusItem *item, struct NSMenu *menu);

void menuet_ns_menu_set_autoenables_items(struct NSMenu *menu, bool enabled);
void menuet_ns_menu_item_set_enabled(struct NSMenuItem *item, bool enabled);
bool menuet_ns_menu_item_enabled(struct NSMenuItem *item);
NSInteger menuet_ns_menu_item_state(struct NSMenuItem *item);
void menuet_ns_menu_item_set_state(struct NSMenuItem *item, NSInteger state);
