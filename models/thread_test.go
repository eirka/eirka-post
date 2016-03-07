package models

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestThreadIsValid(t *testing.T) {

	thread := ThreadModel{
		Uid:         1,
		Ib:          0,
		Ip:          "10.0.0.1",
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
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
		assert.Equal(t, err, e.ErrNoComment, "Error should match")
	}

}

func TestThreadValidateInputCommentShort(t *testing.T) {

	var err error

	thread := ThreadModel{
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
		assert.Equal(t, err, e.ErrCommentShort, "Error should match")
	}

}

func TestThreadValidateInputTitleEmpty(t *testing.T) {

	var err error

	thread := ThreadModel{
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
		assert.Equal(t, err, e.ErrNoTitle, "Error should match")
	}

}

func TestThreadValidateInputTitleShort(t *testing.T) {

	var err error

	thread := ThreadModel{
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
		assert.Equal(t, err, e.ErrTitleShort, "Error should match")
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
		WithArgs(7, "test.jpg", "tests.jpg", "test", 1000, 1000, 100, 100).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	thread := ThreadModel{
		Uid:         1,
		Ib:          1,
		Ip:          "10.0.0.1",
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
	assert.NoError(t, err, "An error was not expected")

}

func TestThreadPostInvalid(t *testing.T) {

	var err error

	thread := ThreadModel{
		Uid:         1,
		Ib:          0,
		Ip:          "10.0.0.1",
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
		assert.Equal(t, err, errors.New("ThreadModel is not valid"), "Error should match")
	}

}
