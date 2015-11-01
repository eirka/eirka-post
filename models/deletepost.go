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

type DeletePostModel struct {
	Thread  uint
	Id      uint
	Ib      uint
	Name    string
	Deleted bool
}

type PostImage struct {
	Id    uint
	File  string
	Thumb string
}

// Status will return info
func (i *DeletePostModel) Status() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// get thread ib and title
	err = db.QueryRow(`SELECT ib_id, thread_title, post_deleted FROM threads 
	INNER JOIN posts on threads.thread_id = posts.thread_id
	WHERE threads.thread_id = ? LIMIT 1`, i.Thread).Scan(&i.Ib, &i.Name, &i.Deleted)
	if err == sql.ErrNoRows {
		return e.ErrNotFound
	} else if err != nil {
		return
	}

	return

}

// Delete will remove the entry
func (i *DeletePostModel) Delete() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	ps1, err := db.Prepare(`UPDATE posts SET post_deleted = ?
	WHERE posts.thread_id = ? AND posts.post_num = ? LIMIT 1`)
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(!i.Deleted, i.Thread, i.Id)
	if err != nil {
		return
	}

	return

}
