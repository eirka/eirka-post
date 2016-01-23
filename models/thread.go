package models

import (
	"errors"
	"github.com/microcosm-cc/bluemonday"
	"html"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

type ThreadModel struct {
	Uid         uint
	Ib          uint
	Ip          string
	Title       string
	Comment     string
	Filename    string
	Thumbnail   string
	MD5         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
}

// check struct validity
func (t *ThreadModel) IsValid() bool {

	if t.Uid == 0 {
		return false
	}

	if t.Ib == 0 {
		return false
	}

	if t.Ip == "" {
		return false
	}

	if t.Title == "" {
		return false
	}

	if t.Comment == "" {
		return false
	}

	if t.Filename == "" {
		return false
	}

	if t.Thumbnail == "" {
		return false
	}

	if t.MD5 == "" {
		return false
	}

	if t.OrigWidth == 0 {
		return false
	}

	if t.OrigHeight == 0 {
		return false
	}

	if t.ThumbWidth == 0 {
		return false
	}

	if t.ThumbHeight == 0 {
		return false
	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (i *ThreadModel) ValidateInput() (err error) {

	if i.Ib == 0 {
		return e.ErrInvalidParam
	}

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize html and xss
	i.Title = html.UnescapeString(p.Sanitize(i.Title))

	// Validate title input
	title := validate.Validate{Input: i.Title, Max: config.Settings.Limits.TitleMaxLength, Min: config.Settings.Limits.TitleMinLength}
	if title.IsEmpty() {
		return e.ErrNoTitle
	} else if title.MinLength() {
		return e.ErrTitleShort
	} else if title.MaxLength() {
		return e.ErrTitleLong
	}

	// sanitize html and xss
	i.Comment = html.UnescapeString(p.Sanitize(i.Comment))

	// Validate comment input
	comment := validate.Validate{Input: i.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}
	if comment.IsEmpty() {
		return e.ErrNoComment
	} else if comment.MaxLength() {
		return e.ErrCommentLong
	}

	return

}

// Post will add the thread to the database with a transaction
func (i *ThreadModel) Post() (err error) {

	// check model validity
	if !i.IsValid() {
		return errors.New("ThreadModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Insert data into threads table
	ps1, err := tx.Prepare("INSERT INTO threads (ib_id,thread_title,thread_first_post,thread_last_post) VALUES (?,?,NOW(),NOW())")
	if err != nil {
		return
	}
	defer ps1.Close()

	// Insert data into posts table
	ps2, err := tx.Prepare("INSERT INTO posts (thread_id,user_id,post_time,post_ip,post_text) VALUES (?,?,NOW(),?,?)")
	if err != nil {
		return
	}
	defer ps2.Close()

	// Insert data into images table
	ps3, err := tx.Prepare("INSERT INTO images (post_id,image_file,image_thumbnail,image_hash,image_orig_height,image_orig_width,image_tn_height,image_tn_width) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		return
	}
	defer ps3.Close()

	e1, err := ps1.Exec(i.Ib, i.Title)
	if err != nil {
		return
	}

	// Get new thread id
	t_id, err := e1.LastInsertId()
	if err != nil {
		return
	}

	e2, err := ps2.Exec(t_id, i.Uid, i.Ip, i.Comment)
	if err != nil {
		return
	}

	// Get new post id
	p_id, err := e2.LastInsertId()
	if err != nil {
		return
	}

	_, err = ps3.Exec(p_id, i.Filename, i.Thumbnail, i.MD5, i.OrigHeight, i.OrigWidth, i.ThumbHeight, i.ThumbWidth)
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
