#import "plugin.h"
#import "PACmanPlugin.h"
#import "../core/core.h"
#include <stdlib.h>
#include <string.h>

menuet_plugin_array menuet_plugins_load(char **message) {
	__block menuet_plugin_array result = {};
	menuet_on_main(^{
		NSURL *directory = NSBundle.mainBundle.builtInPlugInsURL;
		if (!directory) {
			return;
		}

		NSError *error;
		NSArray<NSURL *> *urls = [NSFileManager.defaultManager
			contentsOfDirectoryAtURL:directory
			includingPropertiesForKeys:nil
			options:NSDirectoryEnumerationSkipsHiddenFiles
			error:&error];
		if (!urls) {
			if (![error.domain isEqualToString:NSCocoaErrorDomain] || error.code != NSFileReadNoSuchFileError) {
				*message = strdup(error.localizedDescription.UTF8String);
			}
			return;
		}

		NSMutableArray<NSObject<PACmanPlugin> *> *plugins = [NSMutableArray new];
		for (NSURL *url in [urls sortedArrayUsingComparator:^NSComparisonResult(NSURL *a, NSURL *b) {
			return [a.lastPathComponent compare:b.lastPathComponent];
		}]) {
			if (![url.pathExtension isEqualToString:@"bundle"]) {
				continue;
			}
			NSBundle *bundle = [NSBundle bundleWithURL:url];
			if (!bundle || ![bundle loadAndReturnError:&error]) {
				*message = strdup([NSString stringWithFormat:@"load plugin %@: %@",
					url.lastPathComponent, error.localizedDescription ?: @"invalid bundle"].UTF8String);
				return;
			}
			Class principalClass = bundle.principalClass;
			if (![principalClass conformsToProtocol:@protocol(PACmanPlugin)]) {
				*message = strdup([NSString stringWithFormat:@"plugin %@ does not conform to PACmanPlugin",
					url.lastPathComponent].UTF8String);
				return;
			}
			NSObject<PACmanPlugin> *plugin = [principalClass new];
			if (!plugin) {
				*message = strdup([NSString stringWithFormat:@"initialize plugin %@ failed",
					url.lastPathComponent].UTF8String);
				return;
			}
			[plugins addObject:plugin];
		}

		result.count = plugins.count;
		NSObject * __strong *items = (NSObject * __strong *)calloc(result.count, sizeof(*result.items));
		result.items = (struct NSObject **)(void *)items;
		for (NSUInteger index = 0; index < result.count; index++) {
			items[index] = plugins[index];
		}
	});
	return result;
}

menuet_plugin_menu_item_array menuet_plugin_menu_items(struct NSObject *plugin) {
	__block menuet_plugin_menu_item_array result;
	menuet_on_main(^{
		NSArray<NSMenuItem *> *menuItems = ((__bridge NSObject<PACmanPlugin> *)plugin).menuItems;
		result.count = menuItems.count;
		NSMenuItem * __strong *items = (NSMenuItem * __strong *)calloc(result.count, sizeof(*result.items));
		result.items = (struct NSMenuItem **)(void *)items;
		for (NSUInteger index = 0; index < result.count; index++) {
			items[index] = menuItems[index];
		}
	});
	return result;
}
