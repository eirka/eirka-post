package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestThreadIsValid(t *testing.T) {

	thread := ThreadModel{
		UID:         1,
		Ib:          0,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "test",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	assert.False(t, thread.IsValid(), "Should be false")
}

func TestThreadIsValidImageBadStats(t *testing.T) {

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "test",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   0,
		OrigHeight:  0,
		ThumbWidth:  0,
		ThumbHeight: 0,
	}

	assert.False(t, thread.IsValid(), "Should be false")
}

func TestThreadValidateInputCommentEmpty(t *testing.T) {

	var err error

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrNoComment, err, "Error should match")
	}

}

func TestThreadValidateInputCommentShort(t *testing.T) {

	var err error

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "t",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrCommentShort, err, "Error should match")
	}

}

func TestThreadValidateInputTitleEmpty(t *testing.T) {

	var err error

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "",
		Comment:     "cool post bro",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrNoTitle, err, "Error should match")
	}

}

func TestThreadValidateInputTitleShort(t *testing.T) {

	var err error

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a",
		Comment:     "cool post bro",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrTitleShort, err, "Error should match")
	}

}

func TestThreadPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO threads").
		WithArgs(1, "a cool thread").
		WillReturnResult(sqlmock.NewResult(9, 1))

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(9, 1, "10.0.0.1", "test").
		WillReturnResult(sqlmock.NewResult(7, 1))

	mock.ExpectExec("INSERT INTO images").
		WithArgs(7, "test.jpg", "tests.jpg", "test", "test", 1000, 1000, 100, 100).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "test",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		SHA:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestThreadPostRollback(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO threads").
		WithArgs(1, "a cool thread").
		WillReturnError(errors.New("SQL error"))

	mock.ExpectRollback()

	thread := ThreadModel{
		UID:         1,
		Ib:          1,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "test",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("SQL error"), err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestThreadPostInvalid(t *testing.T) {

	var err error

	thread := ThreadModel{
		UID:         1,
		Ib:          0,
		IP:          "10.0.0.1",
		Title:       "a cool thread",
		Comment:     "test",
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = thread.Post()

	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("ThreadModel is not valid"), err, "Error should match")
	}

}
