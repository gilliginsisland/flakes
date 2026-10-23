package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
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

func (pacman *PACMan) Input(ctx context.Context, u *url.URL, input oc.InputType) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if input != oc.InputPassword {
		return "", fmt.Errorf("unsupported authentication input type: %d", input)
	}
	password, _ := u.User.Password()
	if u.Query().Get("token") != "otp" {
		return password, nil
	}
	pacman.mu.Lock()
	var pd *PooledDialer
	for _, candidate := range pacman.pool.dialers {
		// The factory passes its configured URL unchanged. Identity distinguishes
		// identical proxy URLs and rejects a removed/replaced config entry.
		if candidate.URL == u {
			pd = candidate
			break
		}
	}
	pacman.mu.Unlock()
	if pd == nil {
		return "", errors.New("authentication requested for an inactive proxy")
	}
	token, err := pd.input(ctx, pacman.menu.status, "YubiKey OTP")
	if err != nil {
		return "", err
	}
	return password + token, nil
}

func (pd *PooledDialer) input(ctx context.Context, status *appkit.StatusItem, prompt string) (string, error) {
	pd.menu.input.view.SetMessageText(fmt.Sprintf("Enter %s for %s", prompt, pd.Label))
	pd.menu.input.SetHidden(false)
	defer func() {
		// Hide first so a queued notification click cannot reopen this prompt.
		pd.menu.input.SetHidden(true)
		pd.menu.CancelTracking()
		pd.menu.input.view.SetStringValue("")
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
		case result := <-pd.menu.input.view.Results():
			if result.Canceled {
				return "", errors.New("auth form user cancelled")
			}
			return result.Value, nil
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
