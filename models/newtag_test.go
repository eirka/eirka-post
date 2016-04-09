package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestNewTagValidateInput(t *testing.T) {

	var err error

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 0,
	}

	err = tag.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrInvalidParam, "Error should match")
	}

}

func TestNewTagValidateInputTagShort(t *testing.T) {

	var err error

	tag := NewTagModel{
		Ib:      1,
		Tag:     "t",
		TagType: 1,
	}

	err = tag.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrTagShort, "Error should match")
	}

}

func TestNewTagValidateInputTagEmpty(t *testing.T) {

	var err error

	tag := NewTagModel{
		Ib:      1,
		Tag:     "",
		TagType: 1,
	}

	err = tag.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrNoTagName, "Error should match")
	}

}

func TestNewTagIsValid(t *testing.T) {

	tag := NewTagModel{
		Ib:      1,
		Tag:     "",
		TagType: 1,
	}

	assert.False(t, tag.IsValid(), "Should be false")

}

func TestNewTagStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`select count\(1\) from tags`).WillReturnRows(statusrows)

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagStatusDuplicate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`select count\(1\) from tags`).WillReturnRows(statusrows)

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrDuplicateTag, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectExec("INSERT into tags").
		WithArgs("test", 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tag := NewTagModel{
		Ib:      1,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestNewTagPostInvalid(t *testing.T) {

	var err error

	tag := NewTagModel{
		Ib:      0,
		Tag:     "test",
		TagType: 1,
	}

	err = tag.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("NewTagModel is not valid"), "Error should match")
	}

}
