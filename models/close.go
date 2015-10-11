package models

import (
	"database/sql"

	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type CloseModel struct {
	Id     uint
	Name   string
	Ib     uint
	Closed bool
}

// Status will return info
func (i *CloseModel) Status() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// Check if favorite is already there
	err = db.QueryRow("SELECT ib_id, thread_title, thread_closed FROM threads WHERE thread_id = ? LIMIT 1", i.Id).Scan(&i.Ib, &i.Name, &i.Closed)
	if err == sql.ErrNoRows {
		return e.ErrNotFound
	} else if err != nil {
		return
	}

	return

}

// Toggle will change the thread status
func (i *CloseModel) Toggle() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare("UPDATE threads SET thread_closed = ? WHERE thread_id = ?")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(!i.Closed, i.Id)
	if err != nil {
		return
	}

	return

}
