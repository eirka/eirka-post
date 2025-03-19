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

// Status checks if the image exists and if it's already favorited
// Returns e.ErrFavoriteRemoved if the favorite was removed
// Returns e.ErrNotFound if the image doesn't exist
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

	// Check if image exists
	var imageExists bool
	err = tx.QueryRow("SELECT count(1) FROM images WHERE image_id = ?", m.Image).Scan(&imageExists)
	if err != nil {
		return
	}

	// Return error if image doesn't exist
	if !imageExists {
		return e.ErrNotFound
	}

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

	// Don't commit unless we made changes
	tx.Rollback()

	return
}

// Post adds the favorite to the database
// Verifies the image exists before adding
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

	// Check if image exists
	var imageExists bool
	err = tx.QueryRow("SELECT count(1) FROM images WHERE image_id = ?", m.Image).Scan(&imageExists)
	if err != nil {
		return
	}

	// Return error if image doesn't exist
	if !imageExists {
		return e.ErrNotFound
	}

	// Check if favorite already exists (shouldn't happen with normal flow, but prevents duplicates)
	var favExists bool
	err = tx.QueryRow("SELECT count(1) FROM favorites WHERE image_id = ? AND user_id = ?", m.Image, m.UID).Scan(&favExists)
	if err != nil {
		return
	}

	// Skip if favorite already exists
	if favExists {
		return nil
	}

	// Insert the favorite
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
