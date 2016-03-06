package models

import (
	//"errors"
	"github.com/stretchr/testify/assert"
	//"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	//"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestReplyIsValid(t *testing.T) {

	var err error

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

	var err error

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

	var err error

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

	var err error

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

	var err error

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

	var err error

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
