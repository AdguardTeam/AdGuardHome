package configmgr

import (
	"fmt"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

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
