package appkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#import "alert.h"
*/
import "C"

import (
	"context"
	"errors"
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type ModalResponse int

const (
	AlertFirstButtonReturn  ModalResponse = C.NSAlertFirstButtonReturn
	AlertSecondButtonReturn ModalResponse = C.NSAlertSecondButtonReturn
)

type Alert struct {
	obj *C.struct_NSAlert
}

func NewAlert() *Alert {
	alert := &Alert{obj: C.menuet_ns_alert_new()}
	runtime.AddCleanup(alert, core.Release, core.ObjectRef(alert.obj))
	return alert
}

func (alert *Alert) SetMessageText(text string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_set_message_text(alert.obj, value)
	return alert
}

func (alert *Alert) SetInformativeText(text string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_set_informative_text(alert.obj, value)
	return alert
}

func (alert *Alert) AddButtonWithTitle(title string) *Alert {
	defer runtime.KeepAlive(alert)
	value := C.CString(title)
	defer C.free(unsafe.Pointer(value))
	C.menuet_ns_alert_add_button_with_title(alert.obj, value)
	return alert
}

// RunModal presents an application-modal alert and waits for a button response.
func (alert *Alert) RunModal() ModalResponse {
	defer runtime.KeepAlive(alert)
	return ModalResponse(C.menuet_ns_alert_run_modal(alert.obj))
}

var authenticationPromptSlot = make(chan struct{}, 1)

// Credentials is the user's response to the native sign-in dialog.
type Credentials struct {
	Username string
	Password string
	Remember bool
}

// beginAuthenticationPrompt serializes modal dialogs and cancels them with ctx.
func beginAuthenticationPrompt(ctx context.Context) (*C.struct_MenuetPasswordCancellation, func(), error) {
	select {
	case authenticationPromptSlot <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-authenticationPromptSlot
		return nil, nil, err
	}
	cancellation := C.menuet_password_cancellation_new()
	cancelled := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { C.menuet_password_cancel(cancellation); close(cancelled) })
	return cancellation, func() {
		if !stop() {
			<-cancelled
		}
		C.free(unsafe.Pointer(cancellation))
		<-authenticationPromptSlot
	}, nil
}

// PromptCredentials uses native username and secure password fields. Username
// can be requested alone for authentication forms that ask for it first.
func PromptCredentials(ctx context.Context, message, username string, needsPassword, editableUsername, rememberDefault bool) (Credentials, error) {
	cancellation, cleanup, err := beginAuthenticationPrompt(ctx)
	if err != nil {
		return Credentials{}, err
	}
	defer cleanup()
	text, user := C.CString(message), C.CString(username)
	defer C.free(unsafe.Pointer(text))
	defer C.free(unsafe.Pointer(user))
	var resultUser *C.char
	save := C.bool(rememberDefault)
	password := C.menuet_credentials_prompt(text, user, C.bool(needsPassword), C.bool(editableUsername), &save, &resultUser, cancellation)
	if password != nil {
		defer C.free(unsafe.Pointer(password))
	}
	if resultUser != nil {
		defer C.free(unsafe.Pointer(resultUser))
	}
	if err := ctx.Err(); err != nil {
		return Credentials{}, err
	}
	if password == nil {
		return Credentials{}, errors.New("sign-in cancelled")
	}
	return Credentials{Username: C.GoString(resultUser), Password: C.GoString(password), Remember: bool(save)}, nil
}

// ChooseAccount selects a saved username, or returns an empty username when the
// user chooses Other Account. Password data is not passed to this dialog.
func ChooseAccount(ctx context.Context, message string, accounts []string) (string, error) {
	cancellation, cleanup, err := beginAuthenticationPrompt(ctx)
	if err != nil {
		return "", err
	}
	defer cleanup()
	text := C.CString(message)
	defer C.free(unsafe.Pointer(text))
	names := make([]*C.char, len(accounts))
	for i, account := range accounts {
		names[i] = C.CString(account)
		defer C.free(unsafe.Pointer(names[i]))
	}
	index := int(C.menuet_account_prompt(text, (**C.char)(unsafe.Pointer(unsafe.SliceData(names))), C.size_t(len(names)), cancellation))
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if index < 0 {
		return "", errors.New("account selection cancelled")
	}
	if index >= len(accounts) {
		return "", nil
	}
	return accounts[index], nil
}
