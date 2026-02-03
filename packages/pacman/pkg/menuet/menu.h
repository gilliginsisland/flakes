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

// Add struct definitions for StatusItem to support multiple status items
typedef struct StatusItem {
    char* unique;
    char* title;
    char* imageName;
    MenuItem* submenu;
} StatusItem;

// Add functions for creating, updating, removing, and destroying StatusItem structs
StatusItem* make_status_item();
void destroy_status_item(StatusItem* item);
void update_status_item(StatusItem* item);
void remove_status_item(const char *unique);

@interface MenuetMenu : NSMenu <NSMenuDelegate>

@property(nonatomic, copy) NSString *unique;

@end
