// Package keychain provides Go helpers for passwords stored in macOS Keychain.
package keychain

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
	"github.com/gilliginsisland/pacman/pkg/menuet/security"
)

var ErrPasswordNotFound = errors.New("internet password not found in Keychain")

// InternetPassword identifies an HTTPS server account, shared across VPN
// protocols and paths. Port defaults to 443. Items use an empty path and security
// domain and the default authentication type.
type InternetPassword struct {
	Server  string
	Account string
	Port    int
}

func (item InternetPassword) query() *cf.MutableDictionary {
	port := item.Port
	if port == 0 {
		port = 443
	}
	query := cf.NewMutableDictionary()
	query.SetValue(security.Class, cf.TypeRef(security.ClassInternetPassword))
	query.SetValue(security.AttrProtocol, cf.TypeRef(security.AttrProtocolHTTPS))
	query.SetString(security.AttrPath, "")
	query.SetString(security.AttrSecurityDomain, "")
	query.SetValue(security.AttrAuthenticationType, cf.TypeRef(security.AttrAuthenticationTypeDefault))
	query.SetString(security.AttrServer, strings.ToLower(item.Server))
	if item.Account != "" {
		query.SetString(security.AttrAccount, item.Account)
	}
	query.SetInt(security.AttrPort, port)
	return query
}

// Find reads an existing password, allowing macOS to request access permission.
func (item InternetPassword) Find() (string, error) {
	if item.Account == "" {
		return "", errors.New("Keychain password requires an account")
	}
	query := item.query()
	query.SetValue(security.ReturnData, cf.BooleanTrue)
	query.SetValue(security.MatchLimit, cf.TypeRef(security.MatchLimitOne))
	result, err := security.ItemCopyMatching(&query.Dictionary)
	if errors.Is(err, security.ErrItemNotFound) {
		return "", ErrPasswordNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read Keychain password: %w", err)
	}
	data := result.Data()
	defer clear(data)
	return string(data), nil
}

// Add saves a new password without overwriting an existing server account.
func (item InternetPassword) Add(password string) error {
	if item.Account == "" {
		return errors.New("Keychain password requires an account")
	}
	query := item.query()
	data := []byte(password)
	query.SetData(security.ValueData, data)
	clear(data)
	if err := security.ItemAdd(&query.Dictionary); err != nil {
		return fmt.Errorf("save Keychain password: %w", err)
	}
	return nil
}

// Set replaces a password, or adds it if the server account is not saved yet.
// Call only after successful authentication with the replacement password.
func (item InternetPassword) Set(password string) error {
	if item.Account == "" {
		return errors.New("Keychain password requires an account")
	}
	query := item.query()
	attributes := cf.NewMutableDictionary()
	data := []byte(password)
	attributes.SetData(security.ValueData, data)
	clear(data)
	err := security.ItemUpdate(&query.Dictionary, &attributes.Dictionary)
	if errors.Is(err, security.ErrItemNotFound) {
		return item.Add(password)
	}
	if err != nil {
		return fmt.Errorf("replace Keychain password: %w", err)
	}
	return nil
}

// Accounts discovers matching usernames without requesting password data or
// allowing Keychain authentication UI. An empty Account searches all accounts.
func (item InternetPassword) Accounts() ([]string, error) {
	query := item.query()
	query.SetValue(security.ReturnAttributes, cf.BooleanTrue)
	query.SetValue(security.MatchLimit, cf.TypeRef(security.MatchLimitAll))
	query.SetValue(security.UseAuthenticationUI, cf.TypeRef(security.UseAuthenticationUIFail))
	result, err := security.ItemCopyMatching(&query.Dictionary)
	if errors.Is(err, security.ErrItemNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list Keychain accounts: %w", err)
	}
	var accounts []string
	for _, value := range result.Array() {
		account := value.Dictionary().Get(security.AttrAccount).String()
		if account != "" {
			accounts = append(accounts, account)
		}
	}
	slices.Sort(accounts)
	return slices.Compact(accounts), nil
}
