package models

import (
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type AddTagModel struct {
	Ib    uint
	Tag   uint
	Image uint
	Ip    string
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
	db, err := u.GetDb()
	if err != nil {
		return
	}

	var check bool

	// Check if tag is already there
	err = db.QueryRow("select count(1) from tagmap where tag_id = ? AND image_id = ?", i.Tag, i.Image).Scan(&check)
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

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("INSERT into tagmap (image_id, tag_id, tagmap_ip) VALUES (?,?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Image, i.Tag, i.Ip)
	if err != nil {
		return
	}

	return

}
