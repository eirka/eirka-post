package models

import (
	"database/sql"
	"errors"

	"github.com/asaskevich/govalidator"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// EmailModel contains new email
type EmailModel struct {
	UID          uint
	Name         string
	Email        string
	CurrentEmail string
}

// IsValid will check struct validity
func (m *EmailModel) IsValid() bool {

	if m.UID == 0 || m.UID == 1 {
		return false
	}

	if m.Name == "" {
		return false
	}

	if m.Email == "" {
		return false
	}

	return true

}

// Validate will check the provided email
func (m *EmailModel) Validate() (err error) {

	// Validate email
	if !govalidator.IsEmail(m.Email) {
		return e.ErrInvalidEmail
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// get current email
	err = dbase.QueryRow("SELECT user_name,user_email FROM users WHERE user_id = ?", m.UID).Scan(&m.Name, &m.CurrentEmail)
	if err == sql.ErrNoRows {
		return e.ErrUserNotExist
	} else if err != nil {
		return e.ErrInternalError
	}

	// we dont care if the email is already the same
	if m.Email == m.CurrentEmail {
		return e.ErrEmailSame
	}

	return

}

// Update will update the email model
func (m *EmailModel) Update() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("EmailModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Lock the user row for update
	var check bool
	err = tx.QueryRow("SELECT 1 FROM users WHERE user_id = ? FOR UPDATE", m.UID).Scan(&check)
	if err != nil {
		if err == sql.ErrNoRows {
			return e.ErrUserNotExist
		}
		return
	}

	// Update the user's email
	_, err = tx.Exec("UPDATE users SET user_email = ? WHERE user_id = ?", m.Email, m.UID)
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
