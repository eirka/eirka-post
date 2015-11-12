package models

import (
	"database/sql"

	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
)

type DeletePostModel struct {
	Thread  uint
	Id      uint
	Ib      uint
	Name    string
	Deleted bool
}

// Status will return info
func (i *DeletePostModel) Status() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// get thread ib and title
	err = dbase.QueryRow(`SELECT ib_id, thread_title, post_deleted FROM threads 
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

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// set post to deleted
	ps1, err := tx.Prepare(`UPDATE posts SET post_deleted = ?
	WHERE posts.thread_id = ? AND posts.post_num = ? LIMIT 1`)
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(!i.Deleted, i.Thread, i.Id)
	if err != nil {
		return
	}

	var lasttime string

	// get last post time
	err = tx.QueryRow(`SELECT post_time FROM posts 
	WHERE thread_id = ? AND post_deleted != 1
	ORDER BY post_id DESC LIMIT 1`, i.Thread).Scan(&lasttime)
	if err != nil {
		return
	}

	// update last post time in thread
	ps2, err := tx.Prepare("UPDATE threads SET thread_last_post= ? WHERE thread_id= ?")
	if err != nil {
		return
	}
	defer ps2.Close()

	_, err = ps2.Exec(lasttime, i.Thread)
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
