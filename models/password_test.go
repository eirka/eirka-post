package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestPasswordValidateShortNewPw(t *testing.T) {

	var err error

	password := PasswordModel{
		UID:   1,
		Name:  "test",
		OldPw: "oldpassword",
		NewPw: "short",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordShort, err, "Error should match")
	}

}

func TestPasswordValidateShortOldPw(t *testing.T) {

	var err error

	password := PasswordModel{
		UID:   1,
		Name:  "test",
		OldPw: "short",
		NewPw: "newpassword",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordShort, err, "Error should match")
	}

}

func TestPasswordValidateEmptyNewPw(t *testing.T) {

	var err error

	password := PasswordModel{
		UID:   1,
		Name:  "test",
		OldPw: "oldpassword",
		NewPw: "",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordEmpty, err, "Error should match")
	}

}

func TestPasswordValidateEmptyOldPw(t *testing.T) {

	var err error

	password := PasswordModel{
		UID:   1,
		Name:  "test",
		OldPw: "",
		NewPw: "newpassword",
	}

	err = password.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrPasswordEmpty, err, "Error should match")
	}

}

func TestPasswordIsValid(t *testing.T) {

	password := PasswordModel{
		UID:   1,
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
	defer db.CloseDb()

	mock.ExpectExec("UPDATE users SET user_password").
		WithArgs([]byte("fake"), 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	password := PasswordModel{
		UID:       2,
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
		UID:       1,
		NewHashed: []byte("fake"),
	}

	err = password.Update()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("PasswordModel is not valid"), err, "Error should match")
	}

}
