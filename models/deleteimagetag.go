package models

import (
	"database/sql"

	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
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
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// Check if the tag is there
	err = dbase.QueryRow("SELECT ib_id, tag_name FROM tags WHERE tag_id = ? LIMIT 1", i.Tag).Scan(&i.Ib, &i.Name)
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
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("DELETE FROM tagmap WHERE image_id = ? AND tag_id = ? LIMIT 1")
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
