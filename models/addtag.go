package models

import (
	"errors"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

type AddTagModel struct {
	Ib    uint
	Tag   uint
	Image uint
}

// check struct validity
func (a *AddTagModel) IsValid() bool {

	if a.Ib == 0 {
		return false
	}

	if a.Tag == 0 {
		return false
	}

	if a.Image == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (i *AddTagModel) ValidateInput() (err error) {
	if i.Ib == 0 {
		return e.ErrInvalidParam
	}

	if i.Tag == 0 {
		return e.ErrInvalidParam
	}

	if i.Image == 0 {
		return e.ErrInvalidParam
	}

	return

}

// Status will return info about the thread
func (i *AddTagModel) Status() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	var check bool

	// check to see if this image is in the right ib
	err = dbase.QueryRow(`SELECT count(1) FROM images
	LEFT JOIN posts on images.post_id = posts.post_id
	LEFT JOIN threads on posts.thread_id = threads.thread_id
	WHERE image_id = ? AND ib_id = ?`, i.Image, i.Ib).Scan(&check)
	if err != nil {
		return
	}

	// return if zero
	if !check {
		return e.ErrNotFound
	}

	// Check if tag is already there
	err = dbase.QueryRow("select count(1) from tagmap where tag_id = ? AND image_id = ?", i.Tag, i.Image).Scan(&check)
	if err != nil {
		return
	}

	// return if it does
	if check {
		return e.ErrDuplicateTag
	}

	return

}

// Post will add the reply to the database with a transaction
func (i *AddTagModel) Post() (err error) {

	// check model validity
	if !i.IsValid() {
		return errors.New("AddTagModel is not valid")
	}

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	_, err = dbase.Exec("INSERT into tagmap (image_id, tag_id) VALUES (?,?)", i.Image, i.Tag)
	if err != nil {
		return
	}

	return

}
