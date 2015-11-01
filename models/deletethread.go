package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type DeleteThreadModel struct {
	Id      uint
	Name    string
	Ib      uint
	Deleted bool
}

type ThreadImages struct {
	Id    uint
	File  string
	Thumb string
}

// Status will return info
func (i *DeleteThreadModel) Status() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// Check if favorite is already there
	err = db.QueryRow("SELECT ib_id, thread_title, thread_deleted FROM threads WHERE thread_id = ? LIMIT 1", i.Id).Scan(&i.Ib, &i.Name, &i.Deleted)
	if err == sql.ErrNoRows {
		return e.ErrNotFound
	} else if err != nil {
		return
	}

	return

}

// Delete will remove the entry
func (i *DeleteThreadModel) Delete() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("UPDATE threads SET thread_deleted = ? WHERE thread_id = ?")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(!i.Deleted, i.Id)
	if err != nil {
		return
	}

	return

}
