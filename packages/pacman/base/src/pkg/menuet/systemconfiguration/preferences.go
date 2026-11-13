package systemconfiguration

/*
#include <SystemConfiguration/SystemConfiguration.h>
*/
import "C"

import (
	"fmt"
	"runtime"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
	"github.com/gilliginsisland/pacman/pkg/menuet/security"
)

func configurationError(action string) error {
	return fmt.Errorf("%s: %s", action, C.GoString(C.SCErrorString(C.SCError())))
}

type Preferences struct {
	ref C.SCPreferencesRef
}

func NewPreferencesWithAuthorization(name string, authorization *security.Authorization) (*Preferences, error) {
	text := cf.NewString(name)
	defer C.CFRelease(C.CFTypeRef(text))
	var ref C.SCPreferencesRef
	authorization.WithPointer(func(pointer security.AuthorizationRef) {
		ref = C.SCPreferencesCreateWithAuthorization(0, C.CFStringRef(text), 0, C.AuthorizationRef((*C.struct_AuthorizationOpaqueRef)(pointer)))
	})
	if ref == 0 {
		return nil, configurationError("create network preferences")
	}
	prefs := &Preferences{ref: ref}
	runtime.AddCleanup(prefs, func(ref C.SCPreferencesRef) { C.CFRelease(C.CFTypeRef(ref)) }, ref)
	return prefs, nil
}

// Synchronize discards cached preferences and uncommitted changes. The next
// preferences read reloads the committed values.
func (prefs *Preferences) Synchronize() {
	defer runtime.KeepAlive(prefs)
	C.SCPreferencesSynchronize(prefs.ref)
}

func (prefs *Preferences) Lock(wait bool) error {
	defer runtime.KeepAlive(prefs)
	var flag C.Boolean
	if wait {
		flag = 1
	}
	if C.SCPreferencesLock(prefs.ref, flag) == 0 {
		return configurationError("lock network preferences")
	}
	return nil
}
func (prefs *Preferences) Unlock() {
	defer runtime.KeepAlive(prefs)
	C.SCPreferencesUnlock(prefs.ref)
}
func (prefs *Preferences) CommitChanges() error {
	defer runtime.KeepAlive(prefs)
	if C.SCPreferencesCommitChanges(prefs.ref) == 0 {
		return configurationError("save network preferences")
	}
	return nil
}
func (prefs *Preferences) ApplyChanges() error {
	defer runtime.KeepAlive(prefs)
	if C.SCPreferencesApplyChanges(prefs.ref) == 0 {
		return configurationError("apply network preferences")
	}
	return nil
}
