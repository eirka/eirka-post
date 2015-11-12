package models

import (
	"github.com/asaskevich/govalidator"

	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
)

// EmailModel contains new email
type EmailModel struct {
	Uid   uint
	Email string
}

// Validate will check the provided email
func (r *EmailModel) Validate() (err error) {

	// Validate email
	if !govalidator.IsEmail(r.Email) {
		return e.ErrInvalidEmail
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
