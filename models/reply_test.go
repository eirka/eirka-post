package models

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestReplyIsValid(t *testing.T) {

	reply := ReplyModel{
		Uid:     0,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "hehehe",
		Image:   false,
	}

	assert.False(t, reply.IsValid(), "Should be false")
}

func TestReplyIsValidNoImage(t *testing.T) {

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "a cool comment",
		Image:   false,
	}

	assert.True(t, reply.IsValid(), "Should not be false")
}

func TestReplyIsValidNoCommentNoImage(t *testing.T) {

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "",
		Image:   false,
	}

	assert.False(t, reply.IsValid(), "Should be false")
}

func TestReplyIsValidImage(t *testing.T) {

	reply := ReplyModel{
		Uid:         1,
		Ib:          1,
		Thread:      1,
		Ip:          "10.0.0.1",
		Comment:     "",
		Image:       true,
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
		OrigWidth:   1000,
		OrigHeight:  1000,
		ThumbWidth:  100,
		ThumbHeight: 100,
	}

	assert.True(t, reply.IsValid(), "Should not be false")
}

func TestReplyIsValidImageNoStats(t *testing.T) {

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "",
		Image:   true,
	}

	assert.False(t, reply.IsValid(), "Should be false")
}

func TestReplyIsValidImageBadStats(t *testing.T) {

	reply := ReplyModel{
		Uid:         1,
		Ib:          1,
		Thread:      1,
		Ip:          "10.0.0.1",
		Comment:     "",
		Image:       true,
		Filename:    "",
		Thumbnail:   "",
		MD5:         "",
		OrigWidth:   0,
		OrigHeight:  0,
		ThumbWidth:  0,
		ThumbHeight: 0,
	}

	assert.False(t, reply.IsValid(), "Should be false")
}

func TestReplyValidateInputCommentEmpty(t *testing.T) {

	var err error

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "",
		Image:   false,
	}

	err = reply.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrNoComment, "Error should match")
	}

}

func TestReplyValidateInputCommentShort(t *testing.T) {

	var err error

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "d",
		Image:   false,
	}

	err = reply.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrCommentShort, "Error should match")
	}

}

func TestReplyValidateInputShortCommentWithImage(t *testing.T) {

	var err error

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrCommentShort, "Error should match")
	}

}

func TestReplyStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 0, 2)
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
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

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 1, 100)
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrThreadClosed, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyStatusAutoclose(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	rows := sqlmock.NewRows([]string{"ib", "closed", "total"}).AddRow(1, 1, (config.Settings.Limits.PostsMax + 1))
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).WillReturnRows(rows)

	mock.ExpectExec("UPDATE threads SET thread_closed=1 WHERE thread_id").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "d",
		Image:   true,
	}

	err = reply.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrThreadClosed, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
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

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnResult(sqlmock.NewResult(6, 1))

	mock.ExpectExec("INSERT INTO images").
		WithArgs(6, "test.jpg", "tests.jpg", "test", 1000, 1000, 100, 100).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	reply := ReplyModel{
		Uid:         1,
		Ib:          1,
		Thread:      1,
		Ip:          "10.0.0.1",
		Comment:     "test",
		Image:       true,
		Filename:    "test.jpg",
		Thumbnail:   "tests.jpg",
		MD5:         "test",
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

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(1, 1, "10.0.0.1", "test", 1).
		WillReturnError(errors.New("SQL error"))

	mock.ExpectRollback()

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "10.0.0.1",
		Comment: "test",
		Image:   false,
	}

	err = reply.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("SQL error"), "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestReplyPostInvalid(t *testing.T) {

	var err error

	reply := ReplyModel{
		Uid:     1,
		Ib:      1,
		Thread:  1,
		Ip:      "",
		Comment: "test",
		Image:   false,
	}

	err = reply.Post()

	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("ReplyModel is not valid"), "Error should match")
	}

}
