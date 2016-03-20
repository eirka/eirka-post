package models

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestPasswordValidateShortNewPw(t *testing.T) {

	var err error

	password := PasswordModel{
		Uid:   1,
		Name:  "test",
		OldPw: "oldpassword",
		NewPw: "short",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrPasswordShort, "Error should match")
	}

}

func TestPasswordValidateShortOldPw(t *testing.T) {

	var err error

	password := PasswordModel{
		Uid:   1,
		Name:  "test",
		OldPw: "short",
		NewPw: "newpassword",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrPasswordShort, "Error should match")
	}

}

func TestPasswordValidateEmptyNewPw(t *testing.T) {

	var err error

	password := PasswordModel{
		Uid:   1,
		Name:  "test",
		OldPw: "oldpassword",
		NewPw: "",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrPasswordEmpty, "Error should match")
	}

}

func TestPasswordValidateEmptyOldPw(t *testing.T) {

	var err error

	password := PasswordModel{
		Uid:   1,
		Name:  "test",
		OldPw: "",
		NewPw: "newpassword",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrPasswordEmpty, "Error should match")
	}

}

func TestPasswordIsValid(t *testing.T) {

	password := PasswordModel{
		Uid:   1,
		Name:  "test",
		OldPw: "blah",
		NewPw: "newpassword",
	}

	assert.False(t, password.IsValid(), "Should be false")

}

func TestPasswordUpdate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectExec("UPDATE users SET user_password").
		WithArgs([]byte("fake"), 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	password := PasswordModel{
		Uid:       2,
		Name:      "test",
		OldPw:     "blah",
		NewPw:     "newpassword",
		NewHashed: []byte("fake"),
	}

	err = password.Update()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestPasswordUpdateInvalid(t *testing.T) {

	var err error

	password := PasswordModel{
		Uid:       1,
		NewHashed: []byte("fake"),
	}

	err = password.Update()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("PasswordModel is not valid"), "Error should match")
	}

}
