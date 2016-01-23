package models

import (
	"errors"
	"github.com/microcosm-cc/bluemonday"
	"html"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

type NewTagModel struct {
	Ib      uint
	Tag     string
	TagType uint
}

// check struct validity
func (n *NewTagModel) IsValid() bool {

	if n.Ib == 0 {
		return false
	}

	if n.Tag == "" {
		return false
	}

	if n.TagType == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (i *NewTagModel) ValidateInput() (err error) {
	if i.Ib == 0 {
		return e.ErrInvalidParam
	}

	if i.TagType == 0 {
		return e.ErrInvalidParam
	}

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize for html and xss
	i.Tag = html.UnescapeString(p.Sanitize(i.Tag))

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

	// check model validity
	if !i.IsValid() {
		return errors.New("NewTagModel is not valid")
	}

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
