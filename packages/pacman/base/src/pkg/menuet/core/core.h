#import <Foundation/Foundation.h>
#include <stdint.h>

struct NSObject;
struct NSString;

void *objc_retain(void *object);
void objc_release(void *object);

// Runs the block on the main thread and waits for completion.
void menuet_on_main(void (^block)(void));
struct MenuetCallback *menuet_callback_new(void);
SEL menuet_selector(const char *name);
struct NSString *menuet_ns_string_new(const char *value);
char *menuet_ns_string_value(struct NSString *string);
void menuet_core_invoke_callback(uintptr_t callback, void *sender);
