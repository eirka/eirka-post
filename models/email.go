package models

import (
	"database/sql"
	"github.com/asaskevich/govalidator"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// EmailModel contains new email
type EmailModel struct {
	Uid          uint
	Name         string
	Email        string
	CurrentEmail string
}

// Validate will check the provided email
func (r *EmailModel) Validate() (err error) {

	// Validate email
	if !govalidator.IsEmail(r.Email) {
		return e.ErrInvalidEmail
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// get current email
	err = dbase.QueryRow("SELECT user_name,user_email FROM users WHERE user_id = ?", r.Uid).Scan(&r.Name, &r.CurrentEmail)
	if err == sql.ErrNoRows {
		return e.ErrUserNotExist
	} else if err != nil {
		return e.ErrInternalError
	}

	// we dont care if the email is already the same
	if r.Email == r.CurrentEmail {
		return e.ErrEmailSame
	}

	return

}

// update email
func (r *EmailModel) Update() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("UPDATE users SET user_email = ? WHERE user_id = ?")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(r.Email, r.Uid)
	if err != nil {
		return
	}

	return

}
