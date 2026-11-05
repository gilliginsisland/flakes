// Package authenticationservices wraps system browser authentication APIs using
// the application's existing AppKit run loop.
package authenticationservices

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AuthenticationServices -framework Cocoa
#include <stdlib.h>
#import "session.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/core"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var completions syncutil.Map[
	*C.struct_ASWebAuthenticationSession,
	func(string, error),
]

// Session wraps ASWebAuthenticationSession. Call Cancel when authentication ends.
type Session struct {
	obj *C.struct_ASWebAuthenticationSession
}

// NewSession creates a session without starting it. An empty callbackURLScheme
// accepts no custom callback scheme. Completion runs in a goroutine.
func NewSession(url string, callbackURLScheme string, completion func(callbackURL string, err error)) *Session {
	target := C.CString(url)
	defer C.free(unsafe.Pointer(target))
	scheme := C.CString(callbackURLScheme)
	defer C.free(unsafe.Pointer(scheme))
	obj := C.menuet_as_web_authentication_session_new(target, scheme)
	if obj == nil {
		return nil
	}
	completions.Store(obj, completion)
	session := Session{obj}
	runtime.AddCleanup(&session, func(obj *C.struct_ASWebAuthenticationSession) {
		// Keep the native reference until its callback mapping has been removed.
		completions.Delete(obj)
		C.menuet_as_web_authentication_session_cancel(obj)
		core.Release(unsafe.Pointer(obj))
	}, obj)
	return &session
}

//export menuet_as_web_authentication_session_did_complete
func menuet_as_web_authentication_session_did_complete(
	ptr *C.struct_ASWebAuthenticationSession,
	callbackURL, message *C.char,
) {
	completion, ok := completions.LoadAndDelete(ptr)
	if !ok {
		return
	}
	var err error
	if message != nil {
		err = errors.New(C.GoString(message))
	}
	value := C.GoString(callbackURL)
	go completion(value, err)
}

func (session *Session) Start() bool {
	defer runtime.KeepAlive(session)
	return bool(C.menuet_as_web_authentication_session_start(session.obj))
}

func (session *Session) Cancel() {
	defer runtime.KeepAlive(session)
	C.menuet_as_web_authentication_session_cancel(session.obj)
}

// SetPresentationContextProvider assigns a weak native reference. The caller must
// keep the provider's Go wrapper alive while the session uses it.
func (session *Session) SetPresentationContextProvider(provider unsafe.Pointer) *Session {
	defer runtime.KeepAlive(session)
	C.menuet_as_web_authentication_session_set_presentation_context_provider(session.obj, provider)
	return session
}

func (session *Session) SetPrefersEphemeralWebBrowserSession(ephemeral bool) *Session {
	defer runtime.KeepAlive(session)
	C.menuet_as_web_authentication_session_set_prefers_ephemeral_browser_session(session.obj, C.bool(ephemeral))
	return session
}

// WithPointer calls fn synchronously while keeping the session alive.
// A nil session passes a nil pointer to fn.
func (session *Session) WithPointer(fn func(unsafe.Pointer)) {
	defer runtime.KeepAlive(session)
	var pointer unsafe.Pointer
	if session != nil {
		pointer = unsafe.Pointer(session.obj)
	}
	fn(pointer)
}
