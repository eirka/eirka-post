package models

import (
	"database/sql"
	"github.com/asaskevich/govalidator"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

// loginmodel contains user name and password
type LoginModel struct {
	Name      string
	Password  string
	Hash      []byte
	Id        uint
	Group     uint
	Confirmed bool
	Locked    bool
	Banned    bool
	Sid       string
}

// Validate will check the provided name and password
func (r *LoginModel) Validate() (err error) {

	// Validate name input
	name := u.Validate{Input: r.Name, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if name.IsEmpty() {
		return e.ErrNameEmpty
	} else if name.MinLength() {
		return e.ErrNameShort
	} else if name.MaxLength() {
		return e.ErrNameLong
	} else if !name.IsUsername() {
		return e.ErrNameAlphaNum
	}

	// Validate password input
	password := u.Validate{Input: r.Password, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if password.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if password.MinLength() {
		return e.ErrPasswordShort
	} else if password.MaxLength() {
		return e.ErrPasswordLong
	}

	return

}

// log user in
func (r *LoginModel) Query() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return e.ErrInternalError
	}

	// get hashed password from database
	err = db.QueryRow("select user_id, user_password, usergroup_id, user_confirmed, user_locked, user_banned from users where user_name = ?", r.Name).Scan(&r.Id, &r.Hash, &r.Group, &r.Confirmed, &r.Locked, &r.Banned)
	if err == sql.ErrNoRows {
		return e.ErrInvalidUser
	} else if err != nil {
		return e.ErrInternalError
	}

	return

}
