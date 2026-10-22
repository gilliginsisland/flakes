#import "plugin.h"
#import "PACmanPlugin.h"
#import "../../../pkg/menuet/core/core.h"
#include <stdlib.h>
#include <string.h>

struct NSObject *pacman_plugin_new(void *principalClass, char **message) {
	__block NSObject<PACmanPlugin> *result;
	menuet_on_main(^{
		Class cls = (__bridge Class)principalClass;
		if (![cls conformsToProtocol:@protocol(PACmanPlugin)]) {
			*message = strdup("principal class does not conform to PACmanPlugin");
			return;
		}
		result = [cls new];
		if (!result) {
			*message = strdup("initialize plugin failed");
		}
	});
	return (__bridge_retained struct NSObject *)result;
}

pacman_plugin_menu_item_array pacman_plugin_menu_items(struct NSObject *plugin) {
	__block pacman_plugin_menu_item_array result;
	menuet_on_main(^{
		NSArray<NSMenuItem *> *menuItems = ((__bridge NSObject<PACmanPlugin> *)plugin).menuItems;
		result.count = menuItems.count;
		result.items = calloc(result.count, sizeof(*result.items));
		for (NSUInteger index = 0; index < result.count; index++) {
			result.items[index] = (__bridge_retained struct NSMenuItem *)menuItems[index];
		}
	});
	return result;
}
