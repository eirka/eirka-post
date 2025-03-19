package models

import (
	"errors"
	"html"

	"github.com/microcosm-cc/bluemonday"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// NewTagModel holds the request input
type NewTagModel struct {
	Ib      uint
	Tag     string
	TagType uint
}

// IsValid will check struct validity
func (m *NewTagModel) IsValid() bool {

	if m.Ib == 0 {
		return false
	}

	if m.Tag == "" {
		return false
	}

	if m.TagType == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (m *NewTagModel) ValidateInput() (err error) {
	if m.Ib == 0 {
		return e.ErrInvalidParam
	}

	if m.TagType == 0 {
		return e.ErrInvalidParam
	}

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize for html and xss
	m.Tag = html.UnescapeString(p.Sanitize(m.Tag))

	// Validate name input
	tag := validate.Validate{Input: m.Tag, Max: config.Settings.Limits.TagMaxLength, Min: config.Settings.Limits.TagMinLength}
	if tag.IsEmpty() {
		return e.ErrNoTagName
	} else if tag.MinPartsLength() {
		return e.ErrTagShort
	} else if tag.MaxLength() {
		return e.ErrTagLong
	}

	return

}

// Status will return info about the thread
func (m *NewTagModel) Status() (err error) {

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var check bool

	// Check if tag is already there with row locking
	err = tx.QueryRow("SELECT count(1) FROM tags WHERE ib_id = ? AND tag_name = ? FOR UPDATE", m.Ib, m.Tag).Scan(&check)
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
func (m *NewTagModel) Post() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("NewTagModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT into tags (tag_name,ib_id,tagtype_id) VALUES (?,?,?)",
		m.Tag, m.Ib, m.TagType)
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
