package models

import (
	"database/sql"

	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
)

type DeleteTagModel struct {
	Id   uint
	Name string
	Ib   uint
}

// Status will return info
func (i *DeleteTagModel) Status() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// Check if favorite is already there
	err = dbase.QueryRow("SELECT ib_id, tag_name FROM tags WHERE tag_id = ? LIMIT 1", i.Id).Scan(&i.Ib, &i.Name)
	if err == sql.ErrNoRows {
		return e.ErrNotFound
	} else if err != nil {
		return
	}

	return

}

// Delete will remove the entry
func (i *DeleteTagModel) Delete() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	ps1, err := dbase.Prepare("DELETE FROM tags WHERE tag_id= ? LIMIT 1")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Id)
	if err != nil {
		return
	}

	return

}
