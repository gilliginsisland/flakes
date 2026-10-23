#import "category.h"

struct UNNotificationCategory *menuet_un_category_new(const char *identifier, struct UNNotificationAction **actions, size_t count, UNNotificationCategoryOptions options) {
	@autoreleasepool {
		NSMutableArray<UNNotificationAction *> *values = [NSMutableArray arrayWithCapacity:count];
		for (size_t index = 0; index < count; index++) {
			[values addObject:(__bridge UNNotificationAction *)actions[index]];
		}
		UNNotificationCategory *result = [UNNotificationCategory
			categoryWithIdentifier:[NSString stringWithUTF8String:identifier]
			actions:values
			intentIdentifiers:@[]
			options:options];
		return (__bridge_retained struct UNNotificationCategory *)result;
	}
}
