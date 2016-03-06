package models

import (
	//"database/sql"
	//"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestFavoritesValidateInput(t *testing.T) {

	var err error

	favorite := FavoritesModel{
		Uid:   1,
		Image: 1,
	}

	err = favorite.ValidateInput()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, e.ErrInvalidParam, "Error should match")
	}

}

func TestFavoritesIsValid(t *testing.T) {

	favorite := FavoritesModel{
		Uid:   1,
		Image: 1,
	}

	assert.False(t, favorite.IsValid(), "Should be false")

}

func TestFavoritesStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites`).WillReturnRows(rows)

	favorite := FavoritesModel{
		Uid:   2,
		Image: 1,
	}

	err = favorite.Status()
	assert.NoError(t, err, "An error was not expected")

}
