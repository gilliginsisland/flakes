#import <Foundation/Foundation.h>
#include <stddef.h>

struct objc_class;
struct NSObject;
struct NSMenuItem;

// The caller owns each retained item and must free the raw array separately.
typedef struct pacman_plugin_menu_item_array {
	struct NSMenuItem **items;
	size_t count;
} pacman_plugin_menu_item_array;

struct NSObject *pacman_plugin_new(struct objc_class *principalClass, char **message);
pacman_plugin_menu_item_array pacman_plugin_menu_items(struct NSObject *plugin);
