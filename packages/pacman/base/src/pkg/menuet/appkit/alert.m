#import "alert.h"
#include <stdatomic.h>
#import "../core/core.h"

struct NSAlert *menuet_ns_alert_new(void) {
	__block NSAlert *result;
	menuet_on_main(^{
		result = [NSAlert new];
	});
	return (__bridge_retained struct NSAlert *)result;
}

void menuet_ns_alert_set_message_text(struct NSAlert *alert, const char *text) {
	menuet_on_main(^{
		((__bridge NSAlert *)alert).messageText = [NSString stringWithUTF8String:text];
	});
}

void menuet_ns_alert_set_informative_text(struct NSAlert *alert, const char *text) {
	menuet_on_main(^{
		((__bridge NSAlert *)alert).informativeText = [NSString stringWithUTF8String:text];
	});
}

void menuet_ns_alert_add_button_with_title(struct NSAlert *alert, const char *title) {
	menuet_on_main(^{
		[(__bridge NSAlert *)alert addButtonWithTitle:[NSString stringWithUTF8String:title]];
	});
}

NSModalResponse menuet_ns_alert_run_modal(struct NSAlert *alert) {
	__block NSModalResponse result;
	menuet_on_main(^{
		[NSApp activateIgnoringOtherApps:YES];
		result = [(__bridge NSAlert *)alert runModal];
	});
	return result;
}

struct MenuetPasswordCancellation { atomic_bool cancelled; };

struct MenuetPasswordCancellation *menuet_password_cancellation_new(void) {
    struct MenuetPasswordCancellation *cancellation = malloc(sizeof(*cancellation));
    atomic_init(&cancellation->cancelled, false);
    return cancellation;
}

void menuet_password_cancel(struct MenuetPasswordCancellation *cancellation) {
    atomic_store(&cancellation->cancelled, true);
}


// Called on the main thread by the authentication dialogs below.
static NSModalResponse menuet_run_auth_alert(NSAlert *alert, struct MenuetPasswordCancellation *cancellation) {
    if (atomic_load(&cancellation->cancelled)) return NSModalResponseCancel;
    [NSApp activateIgnoringOtherApps:YES];
    NSTimer *timer = [NSTimer timerWithTimeInterval:0.1 repeats:YES block:^(NSTimer *timer) {
        if (atomic_load(&cancellation->cancelled) && NSApp.modalWindow == alert.window) [NSApp abortModal];
    }];
    [[NSRunLoop mainRunLoop] addTimer:timer forMode:NSModalPanelRunLoopMode];
    NSModalResponse response = [alert runModal];
    [timer invalidate];
    return atomic_load(&cancellation->cancelled) ? NSModalResponseCancel : response;
}

// NSStackView's alignment rect starts six points before NSAlert's text
// column. A plain accessory container keeps the form's labels and field bezels
// aligned with the message and suppression checkbox.
static NSView *menuet_credentials_form(NSTextField *account, NSSecureTextField *password, bool needs_password) {
    NSStackView *fields = [NSStackView stackViewWithViews:@[[NSTextField labelWithString:@"Username"], account]];
    fields.orientation = NSUserInterfaceLayoutOrientationVertical;
    fields.alignment = NSLayoutAttributeLeading;
    fields.spacing = 6;
    [account.widthAnchor constraintEqualToConstant:320].active = YES;
    if (needs_password) {
        [fields addArrangedSubview:[NSTextField labelWithString:@"Password"]];
        [fields addArrangedSubview:password];
        [password.widthAnchor constraintEqualToConstant:320].active = YES;
    }
    NSSize size = fields.fittingSize;
    const CGFloat inset = 6;
    NSView *container = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, size.width + 2 * inset, size.height)];
    [fields setFrame:NSMakeRect(inset, 0, size.width, size.height)];
    [container addSubview:fields];
    return container;
}

char *menuet_credentials_prompt(const char *message, const char *username, bool needs_password, bool editable_username, bool *remember, char **result_username, struct MenuetPasswordCancellation *cancellation) {
    __block char *result = NULL;
    menuet_on_main(^{
        if (atomic_load(&cancellation->cancelled)) return;
        NSAlert *alert = [NSAlert new];
        alert.messageText = @"VPN Sign In";
        alert.informativeText = [NSString stringWithUTF8String:message];
        [alert addButtonWithTitle:@"Connect"];
        [alert addButtonWithTitle:@"Cancel"];
        alert.showsSuppressionButton = needs_password;
        alert.suppressionButton.title = @"Remember account in Keychain";
        alert.suppressionButton.state = *remember ? NSControlStateValueOn : NSControlStateValueOff;
        NSTextField *account = [NSTextField textFieldWithString:[NSString stringWithUTF8String:username]];
        account.editable = editable_username;
        NSSecureTextField *password = [NSSecureTextField new];
        alert.accessoryView = menuet_credentials_form(account, password, needs_password);
        [alert.window setInitialFirstResponder:account.stringValue.length == 0 ? account : (needs_password ? password : account)];
        if (menuet_run_auth_alert(alert, cancellation) == NSAlertFirstButtonReturn) {
            result = strdup(password.stringValue.UTF8String);
            *result_username = strdup(account.stringValue.UTF8String);
            *remember = needs_password && alert.suppressionButton.state == NSControlStateValueOn;
        }
        password.stringValue = @"";
    });
    return result;
}

NSInteger menuet_account_prompt(const char *message, const char **accounts, size_t count, struct MenuetPasswordCancellation *cancellation) {
    __block NSInteger result = -1;
    menuet_on_main(^{
        if (atomic_load(&cancellation->cancelled)) return;
        NSAlert *alert = [NSAlert new];
        alert.messageText = @"Choose VPN Account";
        alert.informativeText = [NSString stringWithUTF8String:message];
        [alert addButtonWithTitle:@"Continue"];
        [alert addButtonWithTitle:@"Cancel"];
        NSPopUpButton *choices = [[NSPopUpButton alloc] initWithFrame:NSMakeRect(0, 0, 320, 26) pullsDown:NO];
        for (size_t i = 0; i < count; ++i) [choices addItemWithTitle:[NSString stringWithUTF8String:accounts[i]]];
        [choices.menu addItem:[NSMenuItem separatorItem]];
        [choices addItemWithTitle:@"Other Account…"];
        alert.accessoryView = choices;
        if (menuet_run_auth_alert(alert, cancellation) == NSAlertFirstButtonReturn) result = choices.indexOfSelectedItem;
    });
    return result;
}
