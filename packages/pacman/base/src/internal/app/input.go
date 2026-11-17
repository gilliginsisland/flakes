package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
	"github.com/gilliginsisland/pacman/pkg/keychain"
	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/notifications"
	"github.com/gilliginsisland/pacman/pkg/notify"
)

var inputNotificationCategory = notifications.NewCategory(notifications.CategoryContent{
	Identifier: "yubi-auth",
	Actions: []*notifications.Action{
		notifications.NewTextInputAction(notifications.TextInputActionContent{
			Identifier:           "yubi-auth-token",
			Title:                "Reply",
			TextInputButtonTitle: "Submit",
			TextInputPlaceholder: "Enter YubiKey OTP",
		}),
	},
	Options: notifications.CategoryOptionCustomDismiss,
})

// Input runs only when the current VPN form requests a username or password.
func (pacman *PACMan) Input(ctx context.Context, u *url.URL, request oc.InputRequest) (oc.InputResult, error) {
	if err := ctx.Err(); err != nil {
		return oc.InputResult{}, err
	}
	result := oc.InputResult{Username: request.Username}
	if result.Username == "" {
		result.Username = u.User.Username()
	}
	password, _ := u.User.Password()
	pacman.mu.Lock()
	var pd *PooledDialer
	for _, candidate := range pacman.pool.dialers {
		if candidate.URL == u {
			pd = candidate
			break
		}
	}
	pacman.mu.Unlock()
	if pd == nil {
		return oc.InputResult{}, errors.New("authentication requested for an inactive proxy")
	}
	if usesKeychain(u) {
		var err error
		result, err = pd.credentials(ctx, request, result.Username)
		if err != nil {
			return oc.InputResult{}, err
		}
	} else {
		// Explicit inline credentials remain supported and take precedence.
		result.Password = password
		if request.NeedUsername && result.Username == "" {
			entered, err := appkit.PromptCredentials(ctx, "Sign in to "+u.Host, "", false, true, false)
			if err != nil {
				return oc.InputResult{}, err
			}
			result.Username = entered.Username
		}
	}
	if request.NeedUsername && result.Username == "" {
		return oc.InputResult{}, errors.New("a VPN username is required")
	}
	if !request.NeedPassword || u.Query().Get("token") != "otp" {
		return result, nil
	}
	pacman.pool.animation.addInput(1)
	defer pacman.pool.animation.addInput(-1)
	token, err := pd.input(ctx, pacman.menu.StatusItem, "YubiKey OTP")
	if err != nil {
		return oc.InputResult{}, err
	}
	result.Password += token
	return result, nil
}

func vpnAccount(u *url.URL, username string) (keychain.InternetPassword, error) {
	port := 443
	if u.Port() != "" {
		var err error
		port, err = strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return keychain.InternetPassword{}, errors.New("invalid VPN port for Keychain credential")
		}
	}
	return keychain.InternetPassword{Server: u.Hostname(), Account: username, Port: port}, nil
}

func (pd *PooledDialer) credentials(ctx context.Context, request oc.InputRequest, username string) (oc.InputResult, error) {
	item, err := vpnAccount(pd.URL, username)
	if err != nil {
		return oc.InputResult{}, err
	}
	if username == "" {
		accounts, err := item.Accounts()
		if err != nil {
			return oc.InputResult{}, err
		}
		switch len(accounts) {
		case 0:
		case 1:
			username = accounts[0]
		default:
			username, err = appkit.ChooseAccount(ctx, "Sign in to "+pd.URL.Host, accounts)
			if err != nil {
				return oc.InputResult{}, err
			}
		}
		item.Account = username
	}
	result := oc.InputResult{Username: username}
	// A username-only form never reads or prompts for a password.
	if !request.NeedPassword && username != "" && !pd.replaceCredentials.Load() {
		return result, nil
	}
	replacing := false
	if request.NeedPassword {
		replacing = pd.replaceCredentials.Swap(false)
	}
	if request.NeedPassword && username != "" && !replacing {
		result.Password, err = item.Find()
		if err == nil {
			return result, ctx.Err()
		}
		if !errors.Is(err, keychain.ErrPasswordNotFound) {
			return oc.InputResult{}, err
		}
	}
	entered, err := appkit.PromptCredentials(ctx, "Sign in to "+pd.URL.Host, username, request.NeedPassword, request.NeedUsername || username == "", replacing)
	if err != nil {
		return oc.InputResult{}, err
	}
	if entered.Username == "" {
		return oc.InputResult{}, errors.New("a VPN username is required")
	}
	result.Username, result.Password = entered.Username, entered.Password
	if request.NeedPassword && entered.Remember {
		item.Account = entered.Username
		result.Accepted = func() error {
			var err error
			if replacing {
				err = item.Set(entered.Password)
			} else {
				err = item.Add(entered.Password)
			}
			if err == nil {
				pd.refreshAccounts()
			}
			return err
		}
	}
	return result, nil
}

func (pd *PooledDialer) input(ctx context.Context, status *appkit.StatusItem, prompt string) (string, error) {
	pd.menu.input.header.SetTitle(fmt.Sprintf("Enter %s for %s", prompt, pd.Label)).
		SetHidden(false)
	pd.menu.input.SetHidden(false)
	defer func() {
		// Hide first so a queued notification click cannot reopen this prompt.
		pd.menu.input.SetHidden(true)
		pd.menu.input.header.SetHidden(true)
		pd.menu.CancelTracking()
		pd.menu.input.SetStringValue("")
	}()

	responses, cleanup := notify.NotifyCh(notify.Notification{
		CategoryIdentifier: "yubi-auth",
		Title:              pd.Label,
		Subtitle:           "Authentication Required",
		Body:               "Enter " + prompt,
		InterruptionLevel:  notifications.InterruptionLevelTimeSensitive,
	})
	defer cleanup()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case result := <-pd.menu.input.Results():
			return result, nil
		case response, ok := <-responses:
			// Notification responses are one-shot. A body click transfers the
			// remaining interaction to the already-visible submenu input item.
			responses = nil
			if !ok {
				return "", errors.New("authentication notification was replaced")
			}
			switch response.ActionIdentifier {
			case notifications.DefaultActionIdentifier:
				pd.menu.input.PopUpMenu(status)
			case notifications.DismissActionIdentifier:
				return "", errors.New("auth form user cancelled")
			case "yubi-auth-token":
				return response.Text, nil
			}
		}
	}
}
