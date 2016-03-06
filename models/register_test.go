package models

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestRegisterValidate(t *testing.T) {

	var err error

	register := RegisterModel{
		Name:     "test",
		Email:    "fart@blah.com",
		Password: "blah",
	}

	err = register.Validate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrInvalidParam, "Error should match")
	}

}

func TestRegisterIsValid(t *testing.T) {

	register := RegisterModel{
		Name:     "test",
		Password: "blah",
		Hashed:   []byte("fake"),
	}

	assert.False(t, tag.IsValid(), "Should be false")

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
		Name:   "test",
		Email:  "test@blah.com",
		Hashed: []byte("fake"),
	}

	err = tag.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.Equal(t, register.Uid, uint(2), "Error should match")

}
