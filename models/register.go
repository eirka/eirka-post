package models

import (
	"github.com/asaskevich/govalidator"
	"golang.org/x/crypto/bcrypt"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

// Register contains information for initial account creation
type RegisterModel struct {
	Name     string
	Email    string
	Password string
}

// Validate will check the provided name length and email
func (r *RegisterModel) Validate() (err error) {

	// Validate name input
	name := u.Validate{Input: r.Name, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if name.IsEmpty() {
		return e.ErrNameEmpty
	} else if name.MinLength() {
		return e.ErrNameShort
	} else if name.MaxLength() {
		return e.ErrNameLong
	}

	// check if name is alphanumeric
	if !govalidator.IsAlphanumeric(r.Name) {
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

	// Validate email
	if !govalidator.IsEmail(r.Email) {
		return e.ErrInvalidEmail
	}

	return

}

// check for duplicate name before registering
func (r *RegisterModel) CheckDuplicate() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	var check bool

	err = db.QueryRow("select count(*) from users where user_name = ?", r.Name).Scan(&check)
	if err != nil {
		return
	}

	// Error if it does
	if check {
		return e.ErrDuplicateName
	}

	return

}

// register new user
func (r *RegisterModel) Register() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// hash password
	password, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	ps1, err := db.Prepare("INSERT into users (usergroup_id, user_name, user_email, user_password, user_confirmed) VALUES (?,?,?,?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(1, r.Name, r.Email, password, 1)
	if err != nil {
		return
	}

	return

}
