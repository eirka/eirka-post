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

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var check bool

	// Check if favorite is already there with row locking to prevent race conditions
	err = tx.QueryRow("SELECT count(1) FROM favorites WHERE image_id = ? AND user_id = ? FOR UPDATE", m.Image, m.UID).Scan(&check)
	if err != nil {
		return
	}

	// delete if it does
	if check {

		_, err = tx.Exec("DELETE FROM favorites WHERE image_id = ? AND user_id = ?", m.Image, m.UID)
		if err != nil {
			return err
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return
		}

		return e.ErrFavoriteRemoved

	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return
	}

	return

}

// Post will add the fav to the database
func (m *FavoritesModel) Post() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("FavoritesModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT into favorites (image_id, user_id) VALUES (?,?)", m.Image, m.UID)
	if err != nil {
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return
	}

	return

}
