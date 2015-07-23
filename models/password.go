package models

import (
	"bytes"
	"github.com/asaskevich/govalidator"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

// Password contains information for initial account creation
type PasswordModel struct {
	Uid       uint
	OldPw     string
	NewPw     string
	OldHashed []byte
	NewHashed []byte
}

// Validate will check the provided password
func (r *PasswordModel) Validate() (err error) {

	// Validate new password input
	password := u.Validate{Input: r.NewPw, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if password.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if password.MinLength() {
		return e.ErrPasswordShort
	} else if password.MaxLength() {
		return e.ErrPasswordLong
	}

	// Validate old password input
	password := u.Validate{Input: r.OldPw, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if password.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if password.MinLength() {
		return e.ErrPasswordShort
	} else if password.MaxLength() {
		return e.ErrPasswordLong
	}

	return

}

// check old password before changing
func (r *PasswordModel) CheckOldPassword() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	var storedpw []byte

	err = db.QueryRow("select user_password from users where user_id = ?", r.Uid).Scan(&storedpw)
	if err != nil {
		return
	}

	// if they arent equal, y'fired
	if !bytes.Equal(r.OldHashed, storedpw) {
		return e.ErrInvalidPassword
	}

	return

}

// change password
func (r *PasswordModel) Update() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("UPDATE users SET user_password = ? WHERE user_id = ?")
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
