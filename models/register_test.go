package models

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

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
		assert.Equal(t, err, e.ErrPasswordEmpty, "Error should match")
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
		assert.Equal(t, err, e.ErrPasswordShort, "Error should match")
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
		assert.Equal(t, err, e.ErrNameEmpty, "Error should match")
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
		assert.Equal(t, err, e.ErrNameShort, "Error should match")
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
		assert.Equal(t, err, e.ErrInvalidEmail, "Error should match")
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

	mock.ExpectBegin()

	mock.ExpectExec("INSERT into users").
		WithArgs("test", "test@blah.com", []byte("fake"), 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectExec("INSERT into user_role_map").
		WithArgs(2, 2).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	register := RegisterModel{
		Name:     "test",
		Password: "blah",
		Email:    "test@blah.com",
		Hashed:   []byte("fake"),
	}

	err = register.Register()
	assert.NoError(t, err, "An error was not expected")

	assert.Equal(t, register.Uid, uint(2), "Error should match")

}

func TestRegisterInvalid(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "",
		Password: "blah",
		Email:    "test@blah.com",
		Hashed:   []byte("fake"),
	}

	err = register.Register()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("RegisterModel is not valid"), "Error should match")
	}

}
