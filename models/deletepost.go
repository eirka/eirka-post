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
	Thread uint
	Id     uint
	Ib     uint
	Name   string
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
	err = db.QueryRow("SELECT ib_id, thread_title FROM threads WHERE thread_id = ? LIMIT 1", i.Thread).Scan(&i.Ib, &i.Name)
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

	image := PostImage{}

	img := true

	// check if post has an image
	err = db.QueryRow(`SELECT image_id,image_file,image_thumbnail FROM posts 
    INNER JOIN images on posts.post_id = images.post_id 
    WHERE posts.thread_id = ? AND posts.post_num = ? LIMIT 1`, i.Thread, i.Id).Scan(&image.Id, &image.File, &image.Thumb)
	if err == sql.ErrNoRows {
		img = false
	} else if err != nil {
		return
	}

	// delete image file
	if img {

		// filename must exist to prevent deleting the directory ;D
		if image.Thumb == "" {
			return
		}

		if image.File == "" {
			return
		}

		// delete from google cloud storage
		u.DeleteGCS(fmt.Sprintf("src/%s", image.File))
		if err != nil {
			return
		}

		u.DeleteGCS(fmt.Sprintf("thumb/%s", image.Thumb))
		if err != nil {
			return
		}

		os.RemoveAll(filepath.Join(config.Settings.General.ImageDir, image.File))
		os.RemoveAll(filepath.Join(config.Settings.General.ThumbnailDir, image.Thumb))

	}

	// delete thread from database
	ps1, err := db.Prepare("DELETE FROM posts WHERE thread_id= ? AND post_num = ? LIMIT 1")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(i.Thread, i.Id)
	if err != nil {
		return
	}

	var lasttime string

	// get last post time
	err = db.QueryRow("SELECT post_time FROM posts WHERE thread_id = ? ORDER BY post_id DESC LIMIT 1", i.Thread).Scan(&lasttime)
	if err != nil {
		return
	}

	// update last post time in thread
	ps1, err := db.Prepare("UPDATE threads SET thread_last_post= ? WHERE thread_id= ?")
	if err != nil {
		return
	}
	defer ps1.Close()

	_, err = ps1.Exec(lasttime, i.Thread)
	if err != nil {
		return
	}

	return

}
