package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework UserNotifications

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

#import "notification.h"

*/
import "C"

// NotificationCategoryOptions represents UNNotificationCategoryOptions
type NotificationCategoryOptions int

const (
	CategoryOptionNone               NotificationCategoryOptions = 0
	CategoryOptionCustomDismiss      NotificationCategoryOptions = 1 << 0
	CategoryOptionAllowInCarPlay     NotificationCategoryOptions = 1 << 1
	CategoryOptionHiddenPreviewsBody NotificationCategoryOptions = 1 << 2
)

// NotificationInputType represents the type of input for a notification action
type NotificationInputType int

const (
	NotificationInputTypeNone NotificationInputType = 0
	NotificationInputTypeText NotificationInputType = 1
)

// NotificationAction represents a base UNNotificationAction
type NotificationAction struct {
	Identifier string
	Title      string
}

func (n NotificationAction) action() *C.NotificationAction {
	action := C.make_notification_action_node()
	n.apply(action)
	return action
}

func (n NotificationAction) apply(action *C.NotificationAction) {
	*action = C.NotificationAction{
		inputType:  C.int(NotificationInputTypeNone),
		identifier: C.CString(n.Identifier),
		title:      C.CString(n.Title),
	}
}

// NotificationActionText represents a UNTextInputNotificationAction
type NotificationActionText struct {
	NotificationAction
	TextInputButtonTitle string
	TextInputPlaceholder string
}

func (n NotificationActionText) action() *C.NotificationAction {
	action := C.make_notification_action_text_node()
	n.apply(action)
	return &(action.action)
}

func (n NotificationActionText) apply(action *C.NotificationActionText) {
	*action = C.NotificationActionText{
		buttonTitle: C.CString(n.TextInputButtonTitle),
		placeholder: C.CString(n.TextInputPlaceholder),
	}
	n.NotificationAction.apply(&action.action)
	action.action.inputType = C.int(NotificationInputTypeText)
}

// NotificationCategory represents a UNNotificationCategory
type NotificationCategory struct {
	Identifier string
	Actions    []Actioner
	Options    NotificationCategoryOptions
}

// Notification represents a UNNotificationRequest
type Notification struct {
	CategoryIdentifier string // Must match a registered category
	Identifier         string // Unique ID for this notification
	Title              string
	Subtitle           string
	Body               string
}

// NotificationResponse represents the response from a notification action
type NotificationResponse struct {
	NotificationIdentifier string
	ActionIdentifier       string
	Text                   string // Only filled if the action requires text input
}

// SetNotificationCategories registers all added notification categories
func (a *Application) SetNotificationCategories(categories []NotificationCategory) {
	if len(categories) == 0 {
		return
	}
	var head *C.NotificationCategory
	defer C.destroy_notification_category_nodes(head)
	curr := &head
	for _, cat := range categories {
		ccat := C.make_notification_category_node()
		*ccat = C.NotificationCategory{
			identifier: C.CString(cat.Identifier),
			actions:    toNotificationNodeActions(cat.Actions...),
			options:    C.int(cat.Options),
		}
		*curr = ccat
		curr = &(*curr).next
	}
	C.set_notification_categories(head)
}

// Notification shows a notification to the user, tied to a registered category
func (a *Application) Notification(notif Notification) {
	cnotif := C.make_notification()
	defer C.destroy_notification(cnotif)
	*cnotif = C.Notification{
		categoryIdentifier: C.CString(notif.CategoryIdentifier),
		identifier:         C.CString(notif.Identifier),
		title:              C.CString(notif.Title),
		subtitle:           C.CString(notif.Subtitle),
		body:               C.CString(notif.Body),
	}
	C.show_notification(cnotif)
}

//export go_notification_response_received
func go_notification_response_received(resp *C.NotificationResponse) {
	defer C.destroy_notification_response(resp)
	go App().NotificationResponder(NotificationResponse{
		NotificationIdentifier: C.GoString(resp.notificationIdentifier),
		ActionIdentifier:       C.GoString(resp.actionIdentifier),
		Text:                   C.GoString(resp.text),
	})
}

func toNotificationNodeActions(actions ...Actioner) *C.NotificationAction {
	var node *C.NotificationAction
	curr := &node
	for _, action := range actions {
		*curr = action.action()
		curr = &(*curr).next
	}
	return node
}

type Actioner interface {
	action() *C.NotificationAction
}
