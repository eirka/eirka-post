package models

import (
	"errors"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
	"github.com/eirka/eirka-libs/validate"
)

// PasswordModel contains information for initial account creation
type PasswordModel struct {
	UID       uint
	Name      string
	OldPw     string
	NewPw     string
	NewHashed []byte
}

// IsValid will check struct validity
func (m *PasswordModel) IsValid() bool {

	if m.UID == 0 || m.UID == 1 {
		return false
	}

	if m.Name == "" {
		return false
	}

	if m.OldPw == "" {
		return false
	}

	if m.NewPw == "" {
		return false
	}

	if m.NewHashed == nil {
		return false
	}

	return true

}

// Validate will check the provided password
func (m *PasswordModel) Validate() (err error) {

	// Validate new password input
	newpassword := validate.Validate{Input: m.NewPw, Max: config.Settings.Limits.PasswordMaxLength, Min: config.Settings.Limits.PasswordMinLength}
	if newpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if newpassword.MinLength() {
		return e.ErrPasswordShort
	} else if newpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	// Validate old password input
	oldpassword := validate.Validate{Input: m.OldPw, Max: config.Settings.Limits.PasswordMaxLength, Min: config.Settings.Limits.PasswordMinLength}
	if oldpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if oldpassword.MinLength() {
		return e.ErrPasswordShort
	} else if oldpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	return

}

// Update will update the password model
func (m *PasswordModel) Update() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("PasswordModel is not valid")
	}

	// update user password
	user.UpdatePassword(m.NewHashed, m.UID)
	if err != nil {
		return
	}

	return

}
