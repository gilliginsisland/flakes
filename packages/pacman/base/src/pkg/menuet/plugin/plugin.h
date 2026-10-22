#import <Foundation/Foundation.h>
#include <stddef.h>

struct NSObject;
struct NSMenuItem;

typedef struct menuet_plugin_array {
	struct NSObject **items;
	size_t count;
} menuet_plugin_array;

typedef struct menuet_plugin_menu_item_array {
	struct NSMenuItem **items;
	size_t count;
} menuet_plugin_menu_item_array;

menuet_plugin_array menuet_plugins_load(char **message);
menuet_plugin_menu_item_array menuet_plugin_menu_items(struct NSObject *plugin);
