package models

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestReplyIsValid(t *testing.T) {

	badreplies := []ReplyModel{
		{UID: 0, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "test", Image: false},
		{UID: 1, Ib: 0, Thread: 1, IP: "127.0.0.1", Comment: "test", Image: false},
		{UID: 1, Ib: 1, Thread: 0, IP: "127.0.0.1", Comment: "test", Image: false},
		{UID: 1, Ib: 1, Thread: 1, IP: "", Comment: "test", Image: false},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: false},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: ""},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: ""},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: ""},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: "hash", OrigWidth: 0},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: "hash", OrigWidth: 100, OrigHeight: 0},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: "hash", OrigWidth: 100, OrigHeight: 100, ThumbWidth: 0},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: "hash", OrigWidth: 100, OrigHeight: 100, ThumbWidth: 100, ThumbHeight: 0},
	}

	for _, reply := range badreplies {
		assert.False(t, reply.IsValid(), "Should be false")
	}

	goodreplies := []ReplyModel{
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "test", Image: false},
		{UID: 1, Ib: 1, Thread: 1, IP: "127.0.0.1", Comment: "test", Image: true, Filename: "filename.png", Thumbnail: "thumb.png", MD5: "hash", OrigWidth: 100, OrigHeight: 100, ThumbWidth: 100, ThumbHeight: 100},
	}

	for _, reply := range goodreplies {
		assert.True(t, reply.IsValid(), "Should be true")
	}
}

func TestReplyValidate(t *testing.T) {

	badreplies := []ReplyModel{
		{Comment: "", Image: false},
		{Comment: "d", Image: false},
		{Comment: randSeq(2000), Image: false},
		{Comment: "d", Image: true},
		{Comment: randSeq(2000), Image: true},
	}

	for _, reply := range badreplies {
		assert.Error(t, reply.ValidateInput(), "Should return error")
	}

	goodreplies := []ReplyModel{
		{Comment: "hello there", Image: false},
		{Comment: "hello there", Image: true},
		{Comment: "", Image: true},
	}

	for _, reply := range goodreplies {
		assert.NoError(t, reply.ValidateInput(), "Should not return error")
	}
}

func TestReplyStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 0, 2)
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyStatusClosed(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 1, 100)
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrThreadClosed, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyStatusAutoclose(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 0, (config.Settings.Limits.PostsMax + 1))
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	mock.ExpectExec("UPDATE threads SET thread_closed=1 WHERE thread_id").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrThreadClosed, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "10.0.0.1",
		Comment: "test",
		Image:   false,
	}

	err = reply.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPostImage(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnResult(sqlmock.NewResult(6, 1))

	mock.ExpectExec("INSERT INTO images").
		WithArgs(6, "test.jpg", "tests.jpg", "test", "test", 1000, 1000, 100, 100).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	reply := ReplyModel{
		UID:         1,
		Ib:          1,
		Thread:      1,
		IP:          "10.0.0.1",
		Comment:     "test",
		Image:       true,
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		SHA:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	err = reply.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPostRollback(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnError(errors.New("SQL error"))

	mock.ExpectRollback()

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "10.0.0.1",
		Comment: "test",
		Image:   false,
	}

	err = reply.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("SQL error"), err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPostInvalid(t *testing.T) {

	var err error

	reply := ReplyModel{
		UID:     1,
		Ib:      1,
		Thread:  1,
		IP:      "",
		Comment: "test",
		Image:   false,
	}

	err = reply.Post()

	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("ReplyModel is not valid"), err, "Error should match")
	}

}
