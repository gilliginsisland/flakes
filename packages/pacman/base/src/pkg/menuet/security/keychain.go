package security

/*
#cgo LDFLAGS: -framework Security
#include <Security/Security.h>
*/
import "C"

import (
	"fmt"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
)

// Borrowed Security framework constants; no retain or release is required.
var (
	Class                         = cf.StringRef(C.kSecClass)
	ClassInternetPassword         = cf.StringRef(C.kSecClassInternetPassword)
	AttrProtocol                  = cf.StringRef(C.kSecAttrProtocol)
	AttrProtocolHTTPS             = cf.StringRef(C.kSecAttrProtocolHTTPS)
	AttrPath                      = cf.StringRef(C.kSecAttrPath)
	AttrSecurityDomain            = cf.StringRef(C.kSecAttrSecurityDomain)
	AttrAuthenticationType        = cf.StringRef(C.kSecAttrAuthenticationType)
	AttrAuthenticationTypeDefault = cf.StringRef(C.kSecAttrAuthenticationTypeDefault)
	AttrServer                    = cf.StringRef(C.kSecAttrServer)
	AttrAccount                   = cf.StringRef(C.kSecAttrAccount)
	AttrPort                      = cf.StringRef(C.kSecAttrPort)
	ReturnAttributes              = cf.StringRef(C.kSecReturnAttributes)
	UseAuthenticationUI           = cf.StringRef(C.kSecUseAuthenticationUI)
	UseAuthenticationUIFail       = cf.StringRef(C.kSecUseAuthenticationUIFail)
	MatchLimitAll                 = cf.StringRef(C.kSecMatchLimitAll)
	ReturnData                    = cf.StringRef(C.kSecReturnData)
	MatchLimit                    = cf.StringRef(C.kSecMatchLimit)
	MatchLimitOne                 = cf.StringRef(C.kSecMatchLimitOne)
	ValueData                     = cf.StringRef(C.kSecValueData)
)

// Status is a native Security framework error code.
type Status int32

const (
	ErrItemNotFound  Status = C.errSecItemNotFound
	ErrDuplicateItem Status = C.errSecDuplicateItem
)

func (status Status) Error() string {
	return fmt.Sprintf("Security framework: OSStatus %d", status)
}

// ItemCopyMatching owns the result returned by SecItemCopyMatching.
func ItemCopyMatching(query *cf.Dictionary) (*cf.Type, error) {
	var result C.CFTypeRef
	var status C.OSStatus
	query.WithPointer(func(ref cf.DictionaryRef) {
		status = C.SecItemCopyMatching(C.CFDictionaryRef(ref), &result)
	})
	if status != C.errSecSuccess {
		return nil, Status(status)
	}
	return cf.OwnType(cf.TypeRef(result)), nil
}

// ItemAdd adds an item without replacing an existing item.
func ItemAdd(attributes *cf.Dictionary) error {
	var status C.OSStatus
	attributes.WithPointer(func(ref cf.DictionaryRef) {
		status = C.SecItemAdd(C.CFDictionaryRef(ref), nil)
	})
	if status != C.errSecSuccess {
		return Status(status)
	}
	return nil
}

// ItemUpdate changes only the supplied attributes of matching items.
func ItemUpdate(query, attributes *cf.Dictionary) error {
	var status C.OSStatus
	query.WithPointer(func(queryRef cf.DictionaryRef) {
		attributes.WithPointer(func(attributesRef cf.DictionaryRef) {
			status = C.SecItemUpdate(C.CFDictionaryRef(queryRef), C.CFDictionaryRef(attributesRef))
		})
	})
	if status != C.errSecSuccess {
		return Status(status)
	}
	return nil
}
