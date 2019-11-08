package models

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestEmailIsValid(t *testing.T) {

	email := EmailModel{
		UID:   1,
		Name:  "test",
		Email: "cool@test.com",
	}

	assert.False(t, email.IsValid(), "Should be false")

}

func TestEmailValidate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"name", "email"}).AddRow("test", "old@test.com")
	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnRows(rows)

	email := EmailModel{
		UID:   2,
		Email: "cool@test.com",
	}

	err = email.Validate()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, email.Name, "test", "Should match")
		assert.Equal(t, email.CurrentEmail, "old@test.com", "Should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestEmailValidateBadEmails(t *testing.T) {

	var err error

	first := EmailModel{
		Email: "notanemail",
	}

	err = first.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInvalidEmail, err, "Error should match")
	}

	second := EmailModel{
		Email: "not@anemail",
	}

	err = second.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInvalidEmail, err, "Error should match")
	}

}

func TestEmailValidateNoUser(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnError(sql.ErrNoRows)

	email := EmailModel{
		UID:   2,
		Email: "cool@test.com",
	}

	err = email.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrUserNotExist, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestEmailValidateSameEmail(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"name", "email"}).AddRow("test", "cool@test.com")
	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnRows(rows)

	email := EmailModel{
		UID:   2,
		Email: "cool@test.com",
	}

	err = email.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrEmailSame, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestEmailUpdate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectExec("UPDATE users SET user_email").
		WithArgs("cool@test.com", 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	email := EmailModel{
		UID:   2,
		Name:  "test",
		Email: "cool@test.com",
	}

	err = email.Update()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestEmailUpdateIsValid(t *testing.T) {

	var err error

	email := EmailModel{
		UID:   2,
		Email: "cool@test.com",
	}

	err = email.Update()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("EmailModel is not valid"), err, "Error should match")
	}

}
