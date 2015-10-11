package models

import (
	"database/sql"

	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type DeleteImageTagModel struct {
	Image uint
	Tag   uint
	Name  string
	Ib    uint
}

// Status will return info
func (i *DeleteImageTagModel) Status() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// Check if the tag is there
	err = db.QueryRow("SELECT ib_id, tag_name FROM tags WHERE tag_id = ? LIMIT 1", i.Tag).Scan(&i.Ib, &i.Name)
	if err == sql.ErrNoRows {
		return e.ErrNotFound
	} else if err != nil {
		return
	}

	return

}

// Delete will remove the entry
func (i *DeleteImageTagModel) Delete() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("DELETE FROM tagmap WHERE image_id = ? AND tag_id = ? LIMIT 1")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Image, i.Tag)
	if err != nil {
		return
	}

	return

}
