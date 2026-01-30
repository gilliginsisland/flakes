#import <Cocoa/Cocoa.h>

typedef enum {
	MenuItemTypeRegular,
	MenuItemTypeSeparator,
	MenuItemTypeSectionHeader,
} MenuItemType;

typedef struct MenuItem {
	struct MenuItem* next;
	MenuItemType type;
	char* unique;
} MenuItem;

typedef struct {
	MenuItem item;
	char* text;
} MenuItemSectionHeader;

typedef struct {
	MenuItem item;
	char* text;
	char* imageName;
	int fontSize;
	float fontWeight;
	bool state;
	bool clickable;
	MenuItem* submenu;
} MenuItemRegular;

MenuItem* make_menu_item(MenuItemType type);
void destroy_menu_items(MenuItem* item);

void set_state(const char *cTitle, const char *cImageName);
void menu_changed(MenuItem* head);

@interface MenuetMenu : NSMenu <NSMenuDelegate>

@property(nonatomic, copy) NSString *unique;

@end
