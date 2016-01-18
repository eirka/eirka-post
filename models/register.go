package models

import (
	"errors"
	"github.com/asaskevich/govalidator"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// Register contains information for initial account creation
type RegisterModel struct {
	Name     string
	Email    string
	Password string
	Hashed   []byte
}

// check struct validity
func (r *RegisterModel) IsValid() bool {

	if r.Name == "" {
		return false
	}

	if r.Password == "" {
		return false
	}

	if r.Hashed == nil {
		return false
	}

	return true

}

// Validate will check the provided name length and email
func (r *RegisterModel) Validate() (err error) {

	// Validate name input
	name := validate.Validate{Input: r.Name, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if name.IsEmpty() {
		return e.ErrNameEmpty
	} else if name.MinLength() {
		return e.ErrNameShort
	} else if name.MaxLength() {
		return e.ErrNameLong
	}

	// Validate password input
	password := validate.Validate{Input: r.Password, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if password.IsEmpty() {
		return e.ErrPasswordEmpty
	} else if password.MinLength() {
		return e.ErrPasswordShort
	} else if password.MaxLength() {
		return e.ErrPasswordLong
	}

	// if theres an email validate it
	if r.Email != "" {
		// Validate email
		if !govalidator.IsEmail(r.Email) {
			return e.ErrInvalidEmail
		}
	}
	return

}

// register new user
func (r *RegisterModel) Register() (err error) {

	if r.Hashed == nil {
		return e.ErrPasswordEmpty
	}

	// check model validity
	if !r.IsValid() {
		return errors.New("RegisterModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("INSERT into users (user_name, user_email, user_password, user_confirmed, user_avatar) VALUES (?,?,?,?,ROUND((RAND() * (48-1))+1))")
	if err != nil {
		return
	}
	defer ps1.Close()

	e1, err := ps1.Exec(r.Name, r.Email, r.Hashed, 1)
	if err != nil {
		return
	}

	user_id, err := e1.LastInsertId()
	if err != nil {
		return err
	}

	ps2, err := dbase.Prepare("INSERT into user_role_map VALUES (?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps2.Exec(user_id, 2)
	if err != nil {
		return
	}

	return

}
