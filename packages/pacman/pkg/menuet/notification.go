package menuet

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UserNotifications

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

#ifndef __NOTIFICATION_H__
#import "notification.h"
#endif

*/
import "C"

import (
	"log"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

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
	InputType  NotificationInputType // Type of input for this action
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
	return action
}

func (n NotificationActionText) apply(action *C.NotificationActionText) {
	*action = C.NotificationActionText{
		buttonTitle:  C.CString(n.TextInputButtonTitle),
		cPlaceholder: C.CString(n.TextInputPlaceholder),
	}
	n.NotificationAction.apply(&action.action)
	action.inputType = NotificationInputTypeText
}

// NotificationCategory represents a UNNotificationCategory
type NotificationCategory struct {
	Identifier string
	Name       string
	Actions    []NotificationAction
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
	ActionIdentifier string
	Text             string // Only filled if the action requires text input
}

var (
	notificationResponses = make(chan NotificationResponse, 10)
	categoryRegistrations = syncutil.Map[uintptr, chan<- struct{}]{}
)

// RegisterNotificationCategory registers a notification category with actions
func (a *Application) RegisterNotificationCategory(category NotificationCategory) {
	if !runningInAppBundle() {
		log.Printf("Warning: notification categories won't be registered unless running inside an application bundle")
		return
	}
	ccategory := C.make_notification_category()
	defer C.destroy_notification_category(ccategory)
	*ccategory = C.NotificationCategory{
		identifier: C.CString(category.Identifier),
		name:       C.CString(category.Name),
		actions:    toNotificationNodeActions(category.Actions),
		options:    C.int(category.Options),
	}
	ch := make(chan struct{}, 1)
	categoryRegistrations.Store(uintptr(unsafe.Pointer(ccategory)), ch)
	C.registerNotificationCategory(ccategory)
	<-ch // Wait for completion signal
}

// Notification shows a notification to the user, tied to a registered category
func (a *Application) Notification(notification Notification) {
	if !runningInAppBundle() {
		log.Printf("Warning: notifications won't show up unless running inside an application bundle")
		return
	}
	cnotif := C.make_notification()
	defer C.destroy_notification(cnotif)
	*cnotif = C.Notification{
		categoryIdentifier: C.CString(notification.CategoryIdentifier),
		identifier:         C.CString(notification.Identifier),
		title:              C.CString(notification.Title),
		subtitle:           C.CString(notification.Subtitle),
		body:               C.CString(notification.Body),
	}
	C.showNotification(cnotif)
}

// NotificationResponses returns a channel to receive notification responses
func (a *Application) NotificationResponses() <-chan NotificationResponse {
	return notificationResponses
}

//export notificationResponseReceived
func notificationResponseReceived(actionIdentifier *C.char, text *C.char) {
	response := NotificationResponse{
		ActionIdentifier: C.GoString(actionIdentifier),
		Text:             C.GoString(text),
	}
	notificationResponses <- response
}

//export notificationCategoryRegistered
func notificationCategoryRegistered(category *C.NotificationCategory) {
	if ch, ok := categoryRegistrations.LoadAndDelete(uintptr(unsafe.Pointer(category))); ok {
		close(ch) // Signal completion by closing the channel
	}
}

func toNotificationNodeActions(actions []NotificationAction) *C.NotificationAction {
	var node *C.NotificationAction
	curr := &node
	for _, action := range actions {
		cIdentifier := C.CString(action.Identifier)
		cTitle := C.CString(action.Title)
		cInputType := C.int(action.InputType)
		if action.InputType == NotificationInputTypeText {
			textAction := action.(NotificationActionText)
			cButtonTitle := C.CString(textAction.TextInputButtonTitle)
			cPlaceholder := C.CString(textAction.TextInputPlaceholder)
			*curr = (*C.NotificationAction)(C.make_notification_action_text_node(cIdentifier, cTitle, cButtonTitle, cPlaceholder))
		} else {
			*curr = C.make_notification_action_node(cIdentifier, cTitle, cInputType)
		}
		curr = &(**curr).next
	}
	return node
}

type Action interface {
	action() *C.NotificationAction
}
