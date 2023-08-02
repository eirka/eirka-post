package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
)

func TestThreadIsValid(t *testing.T) {

	badthreads := []ThreadModel{
		{UID: 0, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 0, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 0, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 0, ThumbWidth: 100, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 0, ThumbHeight: 100},
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 0},
	}

	for _, v := range badthreads {
		assert.False(t, v.IsValid(), "Should return false")
	}

	goodthreads := []ThreadModel{
		{UID: 1, Ib: 1, IP: "127.0.0.1", Title: "test", Comment: "test", Filename: "test.jpg", Thumbnail: "test.jpg", MD5: "test", OrigWidth: 1000, OrigHeight: 1000, ThumbWidth: 100, ThumbHeight: 100},
	}

	for _, v := range goodthreads {
		assert.True(t, v.IsValid(), "Should return true")
	}
}

func TestThreadValidateInput(t *testing.T) {

	badthreads := []ThreadModel{
		{Title: ""},
		{Title: "d"},
		{Title: randSeq(2000)},
		{Title: "cool", Comment: ""},
		{Title: "cool", Comment: "f"},
		{Title: "cool", Comment: randSeq(2000)},
	}

	for _, thread := range badthreads {
		assert.Error(t, thread.ValidateInput(), "Should return error")
	}

	goodthreads := []ThreadModel{
		{Title: "hello there", Comment: "general kenobi"},
	}

	for _, thread := range goodthreads {
		assert.NoError(t, thread.ValidateInput(), "Should not return error")
	}

}

func TestThreadPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

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
	defer db.CloseDb()

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
