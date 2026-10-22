#import <Foundation/Foundation.h>

struct NSBundle;

struct NSBundle *menuet_ns_bundle_main(void);
struct NSBundle *menuet_ns_bundle_with_path(const char *path);
char *menuet_ns_bundle_built_in_plugins_path(struct NSBundle *bundle);
bool menuet_ns_bundle_load(struct NSBundle *bundle, char **message);
void *menuet_ns_bundle_principal_class(struct NSBundle *bundle);
