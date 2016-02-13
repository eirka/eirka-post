package models

import (
	"errors"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// Password contains information for initial account creation
type PasswordModel struct {
	Uid       uint
	Name      string
	OldPw     string
	NewPw     string
	NewHashed []byte
}

// check struct validity
func (p *PasswordModel) IsValid() bool {

	if p.Uid == 0 {
		return false
	}

	if p.Name == "" {
		return false
	}

	if p.OldPw == "" {
		return false
	}

	if p.NewPw == "" {
		return false
	}

	if p.NewHashed == nil {
		return false
	}

	return true

}

// Validate will check the provided password
func (r *PasswordModel) Validate() (err error) {

	// Validate new password input
	newpassword := validate.Validate{Input: r.NewPw, Max: config.Settings.Limits.PasswordMaxLength, Min: config.Settings.Limits.PasswordMinLength}
	if newpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if newpassword.MinLength() {
		return e.ErrPasswordShort
	} else if newpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	// Validate old password input
	oldpassword := validate.Validate{Input: r.OldPw, Max: config.Settings.Limits.PasswordMaxLength, Min: config.Settings.Limits.PasswordMinLength}
	if oldpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if oldpassword.MinLength() {
		return e.ErrPasswordShort
	} else if oldpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	return

}

// change password
func (r *PasswordModel) Update() (err error) {

	// check model validity
	if !r.IsValid() {
		return errors.New("PasswordModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	_, err = dbase.Exec("UPDATE users SET user_password = ? WHERE user_id = ?", r.NewHashed, r.Uid)
	if err != nil {
		return
	}

	return

}
