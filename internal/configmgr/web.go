package configmgr

import (
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// WebUsers represents web UI users.
type WebUsers []*WebUser

// type check
var _ validate.Interface = WebUsers(nil)

// Validate implements the [validate.Interface] interface for WebUsers.
func (wus WebUsers) Validate() (err error) {
	var errs []error
	for idx, u := range wus {
		err = u.Validate()
		if err != nil {
			errs = append(errs, fmt.Errorf("at index %d: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}

// ToInternal converts WebUsers to the internal []*aghuser.User representation.
// wus must be valid.
func (wus WebUsers) ToInternal() (u []*aghuser.User) {
	if wus == nil {
		return nil
	}

	for _, wu := range wus {
		u = append(u, wu.ToInternal())
	}

	return u
}

// WebUser represents a user of the web UI.
type WebUser struct {
	// Name represents the login name of the web user.
	Name string `yaml:"name"`

	// PasswordHash is the hashed representation of the web user password.
	PasswordHash string `yaml:"password"`
}

// type check
var _ validate.Interface = (*WebUser)(nil)

// Validate implements the [validate.Interface] interface for *WebUser.
func (wu *WebUser) Validate() (res error) {
	if wu == nil {
		return errors.ErrNoValue
	}

	return errors.Join(
		validate.NotEmpty("name", wu.Name),
		validate.NotEmpty("password", wu.PasswordHash),
	)
}

// ToInternal converts *WebUser to the internal [*aghuser.User] representation.
// wu must be valid.
func (wu *WebUser) ToInternal() (u *aghuser.User) {
	return &aghuser.User{
		Password: aghuser.NewDefaultPassword(wu.PasswordHash),
		Login:    aghuser.Login(wu.Name),
		ID:       aghuser.NewUserID(),
	}
}

// Theme is an enum of all allowed UI themes.
type Theme string

// Allowed [Theme] values.
//
// Keep in sync with client/src/helpers/constants.ts.
const (
	ThemeAuto  Theme = "auto"
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
)

// NewTheme converts a simple string into a [Theme] and makes sure it's valid.
func NewTheme(s string) (t Theme, err error) {
	switch t = Theme(s); t {
	case ThemeAuto, ThemeLight, ThemeDark:
		return t, nil
	default:
		return "", fmt.Errorf(
			"%w: %q, supported: %q",
			errors.ErrBadEnumValue,
			s,
			[]Theme{ThemeAuto, ThemeLight, ThemeDark},
		)
	}
}

// UnmarshalText implements [encoding.TextUnmarshaler] interface for *Theme.
func (t *Theme) UnmarshalText(b []byte) (err error) {
	*t, err = NewTheme(string(b))

	return err
}

// type check
var _ validate.Interface = Theme("")

// Validate implements the [validate.Interface] interface for Theme.
func (t Theme) Validate() (err error) {
	_, err = NewTheme(string(t))

	return err
}
