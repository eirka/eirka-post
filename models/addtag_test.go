package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestAddTagValidateInput(t *testing.T) {

	var err error

	var badinputs = []AddTagModel{
		{Ib: 0, Tag: 1, Image: 1},
		{Ib: 1, Tag: 0, Image: 1},
		{Ib: 1, Tag: 1, Image: 0},
	}

	for _, input := range badinputs {
		err = input.ValidateInput()
		if assert.Error(t, err, "An error was expected") {
			assert.Equal(t, e.ErrInvalidParam, err, "Error should match")
		}
	}

	goodinput := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = goodinput.ValidateInput()
	assert.NoError(t, err, "No error was expected")
}

func TestAddTagIsValid(t *testing.T) {

	var badinputs = []AddTagModel{
		{Ib: 0, Tag: 1, Image: 1},
		{Ib: 1, Tag: 0, Image: 1},
		{Ib: 1, Tag: 1, Image: 0},
	}

	for _, input := range badinputs {
		assert.False(t, input.IsValid(), "Should be false")
	}

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	assert.True(t, tag.IsValid(), "Should be true")

}

func TestAddTagStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	duperows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM tagmap WHERE tag_id = \? AND image_id = \? FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(duperows)
	mock.ExpectCommit()

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagStatusNotFound(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)
	mock.ExpectRollback()

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrNotFound, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagStatusDuplicate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	duperows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM tagmap WHERE tag_id = \? AND image_id = \? FOR UPDATE`).
		WithArgs(1, 1).
		WillReturnRows(duperows)
	mock.ExpectCommit()

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrDuplicateTag, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagStatusTxError(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin().WillReturnError(errors.New("transaction error"))

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Contains(t, err.Error(), "transaction error", "Error should contain the expected message")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestAddTagPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT into tagmap").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagPostTxError(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin().WillReturnError(errors.New("transaction error"))

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Contains(t, err.Error(), "transaction error", "Error should contain the expected message")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestAddTagPostExecError(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT into tagmap").
		WithArgs(1, 1).
		WillReturnError(errors.New("exec error"))
	mock.ExpectRollback()

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Contains(t, err.Error(), "exec error", "Error should contain the expected message")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestAddTagPostCommitError(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT into tagmap").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Contains(t, err.Error(), "commit error", "Error should contain the expected message")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}
