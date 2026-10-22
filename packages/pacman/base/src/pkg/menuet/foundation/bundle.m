#import "bundle.h"
#import "../core/core.h"
#include <string.h>

struct NSBundle *menuet_ns_bundle_main(void) {
	__block NSBundle *result;
	menuet_on_main(^{ result = NSBundle.mainBundle; });
	return (__bridge_retained struct NSBundle *)result;
}

struct NSBundle *menuet_ns_bundle_with_path(const char *path) {
	__block NSBundle *result;
	menuet_on_main(^{ result = [NSBundle bundleWithPath:[NSString stringWithUTF8String:path]]; });
	return (__bridge_retained struct NSBundle *)result;
}

char *menuet_ns_bundle_built_in_plugins_path(struct NSBundle *bundle) {
	__block char *result;
	menuet_on_main(^{
		NSString *path = ((__bridge NSBundle *)bundle).builtInPlugInsPath;
		result = path ? strdup(path.UTF8String) : NULL;
	});
	return result;
}

bool menuet_ns_bundle_load(struct NSBundle *bundle, char **message) {
	__block bool loaded;
	menuet_on_main(^{
		NSError *error;
		loaded = [(__bridge NSBundle *)bundle loadAndReturnError:&error];
		if (!loaded) {
			*message = strdup(error.localizedDescription.UTF8String);
		}
	});
	return loaded;
}

void *menuet_ns_bundle_principal_class(struct NSBundle *bundle) {
	__block Class result;
	menuet_on_main(^{ result = ((__bridge NSBundle *)bundle).principalClass; });
	return (__bridge void *)result;
}
