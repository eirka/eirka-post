package models

import (
	"github.com/techjanitor/pram-libs/config"
	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
	"github.com/techjanitor/pram-libs/validate"
)

type NewTagModel struct {
	Ib      uint
	Tag     string
	TagType uint
}

// ValidateInput will make sure all the parameters are valid
func (i *NewTagModel) ValidateInput() (err error) {
	if i.Ib == 0 {
		return e.ErrInvalidParam
	}

	if i.TagType == 0 {
		return e.ErrInvalidParam
	}

	// Validate name input
	tag := validate.Validate{Input: i.Tag, Max: config.Settings.Limits.TagMaxLength, Min: config.Settings.Limits.TagMinLength}
	if tag.IsEmpty() {
		return e.ErrNoTagName
	} else if tag.MinLength() {
		return e.ErrTagShort
	} else if tag.MaxLength() {
		return e.ErrTagLong
	}

	return

}

// Status will return info about the thread
func (i *NewTagModel) Status() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	var check bool

	// Check if tag is already there
	err = dbase.QueryRow("select count(1) from tags where ib_id = ? AND tag_name = ?", i.Ib, i.Tag).Scan(&check)
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
func (i *NewTagModel) Post() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("INSERT into tags (tag_name,ib_id,tagtype_id) VALUES (?,?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Tag, i.Ib, i.TagType)
	if err != nil {
		return
	}

	return

}
