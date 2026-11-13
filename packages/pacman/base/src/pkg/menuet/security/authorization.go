// Package security wraps macOS Security framework authorization sessions.
package security

/*
#cgo LDFLAGS: -framework Security
#include <Security/Authorization.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

type AuthorizationFlags uint32

const (
	AuthorizationFlagDefaults           AuthorizationFlags = C.kAuthorizationFlagDefaults
	AuthorizationFlagExtendRights       AuthorizationFlags = C.kAuthorizationFlagExtendRights
	AuthorizationFlagInteractionAllowed AuthorizationFlags = C.kAuthorizationFlagInteractionAllowed
)

type Authorization struct{ ref C.AuthorizationRef }

type AuthorizationRef = *C.struct_AuthorizationOpaqueRef

func NewAuthorization() (*Authorization, error) {
	var ref C.AuthorizationRef
	if status := C.AuthorizationCreate(nil, nil, C.kAuthorizationFlagDefaults, &ref); status != C.errAuthorizationSuccess {
		return nil, fmt.Errorf("create authorization session: OSStatus %d", status)
	}
	authorization := &Authorization{ref: ref}
	runtime.AddCleanup(authorization, func(ref C.AuthorizationRef) {
		C.AuthorizationFree(ref, C.kAuthorizationFlagDefaults)
	}, ref)
	return authorization, nil
}

func (authorization *Authorization) CopyRight(right string, flags AuthorizationFlags) error {
	defer runtime.KeepAlive(authorization)
	value := C.CString(right)
	defer C.free(unsafe.Pointer(value))
	item := (*C.AuthorizationItem)(C.malloc(C.sizeof_AuthorizationItem))
	defer C.free(unsafe.Pointer(item))
	*item = C.AuthorizationItem{name: value}
	rights := C.AuthorizationRights{count: 1, items: item}
	if status := C.AuthorizationCopyRights(authorization.ref, &rights, nil, C.AuthorizationFlags(flags), nil); status != C.errAuthorizationSuccess {
		return fmt.Errorf("authorize %s: OSStatus %d", right, status)
	}
	return nil
}

// WithPointer calls fn synchronously while keeping the authorization alive.
// A nil authorization passes a nil pointer to fn.
func (authorization *Authorization) WithPointer(fn func(AuthorizationRef)) {
	defer runtime.KeepAlive(authorization)
	var pointer AuthorizationRef
	if authorization != nil {
		pointer = AuthorizationRef(authorization.ref)
	}
	fn(pointer)
}
