package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

func TestFavoritesValidateInput(t *testing.T) {

	var err error

	badfavorites := []FavoritesModel{
		{UID: 0, Image: 1},
		{UID: 1, Image: 1},
		{UID: 2, Image: 0},
	}

	for _, input := range badfavorites {
		err = input.ValidateInput()
		if assert.Error(t, err, "An error was expected") {
			assert.Equal(t, e.ErrInvalidParam, err, "Error should match")
		}
	}

	goodfavorite := FavoritesModel{
		UID:   2,
		Image: 1,
	}

	err = goodfavorite.ValidateInput()
	assert.NoError(t, err, "An error was not expected")

}

func TestFavoritesIsValid(t *testing.T) {

	badfavorites := []FavoritesModel{
		{UID: 0, Image: 1},
		{UID: 1, Image: 1},
		{UID: 2, Image: 0},
	}

	for _, input := range badfavorites {
		assert.False(t, input.IsValid(), "Should be false")
	}

	goodfavorite := FavoritesModel{
		UID:   2,
		Image: 1,
	}

	assert.True(t, goodfavorite.IsValid(), "Should be true")

}

func TestFavoritesStatus(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites`).WillReturnRows(rows)

	favorite := FavoritesModel{
		UID:   2,
		Image: 1,
	}

	err = favorite.Status()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestFavoritesStatusRemove(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites`).WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM favorites").
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	favorite := FavoritesModel{
		UID:   2,
		Image: 1,
	}

	err = favorite.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrFavoriteRemoved, err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestFavoritesPost(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	mock.ExpectExec("INSERT into favorites").
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	favorite := FavoritesModel{
		UID:   2,
		Image: 1,
	}

	err = favorite.Post()
	assert.NoError(t, err, "An error was not expected")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestFavoritesPostInvalid(t *testing.T) {

	var err error

	favorite := FavoritesModel{
		UID:   1,
		Image: 1,
	}

	err = favorite.Post()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("FavoritesModel is not valid"), err, "Error should match")
	}

}

func TestFavoriteStatusInvalid(t *testing.T) {

	var err error

	favorite := FavoritesModel{
		UID:   1,
		Image: 1,
	}

	err = favorite.Status()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("FavoritesModel is not valid"), err, "Error should match")
	}

}
