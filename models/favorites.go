package models

import (
	"errors"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// FavoritesModel holds the request input
type FavoritesModel struct {
	UID   uint
	Image uint
}

// IsValid will check struct validity
func (m *FavoritesModel) IsValid() bool {

	if m.UID == 0 || m.UID == 1 {
		return false
	}

	if m.Image == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (m *FavoritesModel) ValidateInput() (err error) {

	if m.UID == 0 || m.UID == 1 {
		return e.ErrInvalidParam
	}

	if m.Image == 0 {
		return e.ErrInvalidParam
	}

	return

}

// Status will return info
func (m *FavoritesModel) Status() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("FavoritesModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	var check bool

	// Check if favorite is already there
	err = dbase.QueryRow("SELECT count(1) FROM favorites WHERE image_id = ? AND user_id = ?", m.Image, m.UID).Scan(&check)
	if err != nil {
		return
	}

	// delete if it does
	if check {

		_, err = dbase.Exec("DELETE FROM favorites WHERE image_id = ? AND user_id = ?", m.Image, m.UID)
		if err != nil {
			return err
		}

		return e.ErrFavoriteRemoved

	}

	return

}

// Post will add the fav to the database
func (m *FavoritesModel) Post() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("FavoritesModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	_, err = dbase.Exec("INSERT into favorites (image_id, user_id) VALUES (?,?)", m.Image, m.UID)
	if err != nil {
		return
	}

	return

}
