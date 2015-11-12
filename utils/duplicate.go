package utils

import (
	"github.com/techjanitor/pram-libs/db"
	e "github.com/techjanitor/pram-libs/errors"
)

type CheckDuplicate struct {
	Ib     uint
	MD5    string
	check  bool
	Thread *int
	Post   *int
}

func (c *CheckDuplicate) Get() (err error) {
	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	err = dbase.QueryRow(`select count(1),posts.post_num,threads.thread_id from threads 
	LEFT JOIN posts on threads.thread_id = posts.thread_id 
	LEFT JOIN images on posts.post_id = images.post_id 
	WHERE image_hash = ? AND ib_id = ?`, c.MD5, c.Ib).Scan(&c.check, &c.Post, &c.Thread)
	if err != nil {
		return
	}

	// Delete if it does
	if c.check {
		return e.ErrDuplicateImage
	}

	return
}
