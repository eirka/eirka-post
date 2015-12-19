package models

import (
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
	OldHashed []byte
	NewHashed []byte
}

// Validate will check the provided password
func (r *PasswordModel) Validate() (err error) {

	// Validate new password input
	newpassword := validate.Validate{Input: r.NewPw, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if newpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if newpassword.MinLength() {
		return e.ErrPasswordShort
	} else if newpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	// Validate old password input
	oldpassword := validate.Validate{Input: r.OldPw, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if oldpassword.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if oldpassword.MinLength() {
		return e.ErrPasswordShort
	} else if oldpassword.MaxLength() {
		return e.ErrPasswordLong
	}

	return

}

// check old password before changing
func (r *PasswordModel) GetOldPassword() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// get stored password
	err = dbase.QueryRow("SELECT user_name,user_password FROM users WHERE user_id = ?", r.Uid).Scan(&r.Name, &r.OldHashed)
	if err != nil {
		return
	}

	return

}

// change password
func (r *PasswordModel) Update() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("UPDATE users SET user_password = ? WHERE user_id = ?")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(r.NewHashed, r.Uid)
	if err != nil {
		return
	}

	return

}
