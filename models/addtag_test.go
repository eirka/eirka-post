package models

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/db"
)

func TestAddTagStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	duperows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`select count\(1\) from tagmap`).WillReturnRows(duperows)

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Status()
	assert.NoError(t, err, "An error was not expected")

}

func TestAddTagPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	mock.ExpectExec("INSERT into tagmap").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	tag := AddTagModel{
		Ib:    1,
		Tag:   1,
		Image: 1,
	}

	err = tag.Post()
	assert.NoError(t, err, "An error was not expected")

}