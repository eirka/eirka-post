package models

import (
	"errors"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

type FavoritesModel struct {
	Uid   uint
	Image uint
}

// check struct validity
func (f *FavoritesModel) IsValid() bool {

	if f.Uid == 0 || f.Uid == 1 {
		return false
	}

	if f.Image == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (i *FavoritesModel) ValidateInput() (err error) {

	if i.Uid == 0 || i.Uid == 1 {
		return e.ErrInvalidParam
	}

	if i.Image == 0 {
		return e.ErrInvalidParam
	}

	return

}

// Status will return info
func (i *FavoritesModel) Status() (err error) {

	// check model validity
	if !i.IsValid() {
		return errors.New("FavoritesModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	var check bool

	// Check if favorite is already there
	err = dbase.QueryRow("SELECT count(1) FROM favorites WHERE image_id = ? AND user_id = ?", i.Image, i.Uid).Scan(&check)
	if err != nil {
		return
	}

	// delete if it does
	if check {

		ps1, err := dbase.Prepare("DELETE FROM favorites WHERE image_id = ? AND user_id = ? LIMIT 1")
		if err != nil {
			return err
		}
		defer ps1.Close()

		_, err = ps1.Exec(i.Image, i.Uid)
		if err != nil {
			return err
		}

		return e.ErrFavoriteRemoved

	}

	return

}

// Post will add the fav to the database
func (i *FavoritesModel) Post() (err error) {

	// check model validity
	if !i.IsValid() {
		return errors.New("FavoritesModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("INSERT into favorites (image_id, user_id) VALUES (?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Image, i.Uid)
	if err != nil {
		return
	}

	return

}
