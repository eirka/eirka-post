package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestRegisterValidatePasswordEmpty(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "test",
		Email:    "fart@blah.com",
		Password: "",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordEmpty, err, "Error should match")
	}

}

func TestRegisterValidatePasswordShort(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "test",
		Email:    "fart@blah.com",
		Password: "blah",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordShort, err, "Error should match")
	}

}

func TestRegisterValidateNameEmpty(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "",
		Email:    "fart@blah.com",
		Password: "password",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrNameEmpty, err, "Error should match")
	}

}

func TestRegisterValidateNameShort(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "te",
		Email:    "fart@blah.com",
		Password: "password",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrNameShort, err, "Error should match")
	}

}

func TestRegisterValidateEmail(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "test",
		Email:    "fart@blah",
		Password: "password",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInvalidEmail, err, "Error should match")
	}

}

func TestRegisterValidateEmailMissing(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "test",
		Email:    "",
		Password: "password",
	}

	err = register.Validate()
	assert.NoError(t, err, "An error was not expected")

}

func TestRegisterIsValid(t *testing.T) {

	register := RegisterModel{
		Name:     "",
		Password: "blah",
		Hashed:   []byte("fake"),
	}

	assert.False(t, register.IsValid(), "Should be false")

}

func TestRegister(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT into users").
		WithArgs("test", "test@blah.com", []byte("fake"), 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectExec("INSERT into user_role_map").
		WithArgs(2, 2).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	register := RegisterModel{
		Name:   "test",
		Email:  "test@blah.com",
		Hashed: []byte("fake"),
	}

	err = register.Register()
	assert.NoError(t, err, "An error was not expected")

	assert.Equal(t, register.UID, uint(2), "Error should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestRegisterRollback(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT into users").
		WithArgs("test", "test@blah.com", []byte("fake"), 1).
		WillReturnError(errors.New("SQL error"))

	mock.ExpectRollback()

	register := RegisterModel{
		Name:   "test",
		Email:  "test@blah.com",
		Hashed: []byte("fake"),
	}

	err = register.Register()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("SQL error"), err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestRegisterInvalidName(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:   "",
		Email:  "test@blah.com",
		Hashed: []byte("fake"),
	}

	err = register.Register()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("RegisterModel is not valid"), err, "Error should match")
	}

}

func TestRegisterInvalidHash(t *testing.T) {

	var err error

	register := RegisterModel{
		Name: "derp",
	}

	err = register.Register()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("RegisterModel is not valid"), err, "Error should match")
	}

}
