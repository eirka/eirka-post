package models

import (
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type FavoritesModel struct {
	Uid   uint
	Image uint
	Ip    string
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

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	var check bool

	// Check if favorite is already there
	err = db.QueryRow("SELECT count(1) FROM favorites WHERE image_id = ? AND user_id = ?", i.Image, i.Uid).Scan(&check)
	if err != nil {
		return
	}

	// delete if it does
	if check {

		ps1, err := db.Prepare("DELETE FROM favorites WHERE image_id = ? AND user_id = ? LIMIT 1")
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

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("INSERT into favorites (image_id, user_id) VALUES (?,?)")
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
