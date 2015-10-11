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
	Id   uint
	Name string
	Ib   uint
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
	err = db.QueryRow("SELECT ib_id, thread_title FROM threads WHERE thread_id = ? LIMIT 1", i.Id).Scan(&i.Ib, &i.Name)
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

	images := []ThreadImages{}

	// Get thread images
	rows, err := db.Query(`SELECT image_id,image_file,image_thumbnail FROM images
    INNER JOIN posts on images.post_id = posts.post_id
    INNER JOIN threads on threads.thread_id = posts.thread_id
    WHERE threads.thread_id = ?`, i.Id)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		image := ThreadImages{}

		err := rows.Scan(&image.Id, &image.File, &image.Thumb)
		if err != nil {
			return err
		}
		// Append rows to info struct
		images = append(images, image)
	}
	err = rows.Err()
	if err != nil {
		return
	}

	// delete image files
	go func() {

		for _, image := range images {

			go func() {

				// filename must exist to prevent deleting the directory ;D
				if image.Thumb == "" {
					fmt.Fprintln(os.Stderr, err)
					return
				}

				if image.File == "" {
					fmt.Fprintln(os.Stderr, err)
					return
				}

				// delete from google cloud storage
				u.DeleteGCS(fmt.Sprintf("src/%s", image.File))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}

				u.DeleteGCS(fmt.Sprintf("thumb/%s", image.Thumb))
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
				}

				os.RemoveAll(filepath.Join(config.Settings.General.ImageDir, image.File))
				os.RemoveAll(filepath.Join(config.Settings.General.ThumbnailDir, image.Thumb))

			}()

		}

	}()

	// delete thread from database
	ps1, err := db.Prepare("DELETE FROM threads WHERE thread_id= ? LIMIT 1")
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
