package models

import (
	"errors"
	"html"

	"github.com/asaskevich/govalidator"
	"github.com/microcosm-cc/bluemonday"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// RegisterModel contains information for initial account creation
type RegisterModel struct {
	UID      uint
	Name     string
	Email    string
	Password string
	Hashed   []byte
}

// IsValid will check struct validity
func (r *RegisterModel) IsValid() bool {

	if r.Name == "" {
		return false
	}

	if r.Hashed == nil || len(r.Hashed) == 0 {
		return false
	}

	return true

}

// Validate will check the provided name length and email
func (r *RegisterModel) Validate() (err error) {

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize for html and xss
	r.Name = html.UnescapeString(p.Sanitize(r.Name))

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
	password := validate.Validate{Input: r.Password, Max: config.Settings.Limits.PasswordMaxLength, Min: config.Settings.Limits.PasswordMinLength}
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

// Register will create a new user
func (r *RegisterModel) Register() (err error) {

	// check model validity
	if !r.IsValid() {
		return errors.New("RegisterModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	e1, err := tx.Exec("INSERT into users (user_name, user_email, user_password, user_confirmed) VALUES (?,?,?,?)",
		r.Name, r.Email, r.Hashed, 1)
	if err != nil {
		return
	}

	uid, err := e1.LastInsertId()
	if err != nil {
		return err
	}

	r.UID = uint(uid)

	if r.UID == 0 || r.UID == 1 {
		return errors.New("user ID is invalid")
	}

	_, err = tx.Exec("INSERT into user_role_map VALUES (?,?)", r.UID, 2)
	if err != nil {
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return
	}

	return

}
