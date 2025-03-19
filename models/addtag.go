package models

import (
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// AddTagModel holds the data from the request
type AddTagModel struct {
	Ib    uint
	Tag   uint
	Image uint
}

// IsValid will check struct validity
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
func (a *AddTagModel) ValidateInput() (err error) {
	if a.Ib == 0 {
		return e.ErrInvalidParam
	}

	if a.Tag == 0 {
		return e.ErrInvalidParam
	}

	if a.Image == 0 {
		return e.ErrInvalidParam
	}

	return

}

// Status will return info about the thread
func (a *AddTagModel) Status() (err error) {

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var check bool

	// check to see if this image is in the right ib
	err = tx.QueryRow(`SELECT count(1) FROM images
	LEFT JOIN posts on images.post_id = posts.post_id
	LEFT JOIN threads on posts.thread_id = threads.thread_id
	WHERE image_id = ? AND ib_id = ?`, a.Image, a.Ib).Scan(&check)
	if err != nil {
		return
	}

	// return if zero
	if !check {
		return e.ErrNotFound
	}

	// Check if tag is already there with row locking to prevent race conditions
	err = tx.QueryRow("SELECT count(1) FROM tagmap WHERE tag_id = ? AND image_id = ? FOR UPDATE", a.Tag, a.Image).Scan(&check)
	if err != nil {
		return
	}

	// Commit transaction to release locks
	err = tx.Commit()
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
func (a *AddTagModel) Post() (err error) {

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Insert tag mapping
	_, err = tx.Exec("INSERT into tagmap (image_id, tag_id) VALUES (?,?)", a.Image, a.Tag)
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
