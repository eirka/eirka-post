package models

import (
	//"errors"
	"github.com/stretchr/testify/assert"
	//"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	//"github.com/eirka/eirka-libs/db"
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
