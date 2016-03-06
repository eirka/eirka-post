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

type ReplyModel struct {
	Uid         uint
	Ib          uint
	Thread      uint
	Ip          string
	Comment     string
	Filename    string
	Thumbnail   string
	MD5         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
	Image       bool
}

// check struct validity
func (r *ReplyModel) IsValid() bool {

	if r.Uid == 0 {
		return false
	}

	if r.Ib == 0 {
		return false
	}

	if r.Thread == 0 {
		return false
	}

	if r.Ip == "" {
		return false
	}

	if !r.Image && r.Comment == "" {
		return false
	}

	if r.Image {

		if r.Filename == "" {
			return false
		}

		if r.Thumbnail == "" {
			return false
		}

		if r.MD5 == "" {
			return false
		}

		if r.OrigWidth == 0 {
			return false
		}

		if r.OrigHeight == 0 {
			return false
		}

		if r.ThumbWidth == 0 {
			return false
		}

		if r.ThumbHeight == 0 {
			return false
		}

	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (i *ReplyModel) ValidateInput() (err error) {

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize for html and xss
	i.Comment = html.UnescapeString(p.Sanitize(i.Comment))

	// There must either be a comment, an image, or an image with a comment
	// If theres no image a comment is required
	comment := validate.Validate{Input: i.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}

	// if there is no image check the comment
	if !i.Image {
		if comment.IsEmpty() {
			return e.ErrNoComment
		} else if comment.MinLength() {
			return e.ErrCommentShort
		} else if comment.MaxLength() {
			return e.ErrCommentLong
		}
	}

	// If theres an image and a comment validate comment
	if i.Image && !comment.IsEmpty() {
		if comment.MinLength() {
			return e.ErrCommentShort
		} else if comment.MaxLength() {
			return e.ErrCommentLong
		}
	}

	return

}

// Status will return info about the thread
func (i *ReplyModel) Status() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	var closed bool
	var total uint

	// Check if thread is closed and get the total amount of posts
	err = dbase.QueryRow(`SELECT ib_id,thread_closed,count(post_num) FROM threads
    INNER JOIN posts on threads.thread_id = posts.thread_id
    WHERE threads.thread_id = ? AND post_deleted != 1`, i.Thread).Scan(&i.Ib, &closed, &total)
	if err != nil {
		return
	}

	// Error if thread is closed
	if closed {
		return e.ErrThreadClosed
	}

	// Close thread if above max posts
	if total > config.Settings.Limits.PostsMax {

		_, err = dbase.Exec("UPDATE threads SET thread_closed=1 WHERE thread_id = ?",
			i.Thread)
		if err != nil {
			return err
		}

		return e.ErrThreadClosed
	}

	return

}

// Post will add the reply to the database with a transaction
func (i *ReplyModel) Post() (err error) {

	// check model validity
	if !i.IsValid() {
		return errors.New("ReplyModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// insert new post
	e1, err := tx.Exec(`INSERT INTO posts (thread_id,user_id,post_num,post_time,post_ip,post_text)
    SELECT ?,?,max(post_num)+1,NOW(),?,?
    FROM posts WHERE thread_id = ?`, i.Thread, i.Uid, i.Ip, i.Comment, i.Thread)
	if err != nil {
		return
	}

	if i.Image {

		p_id, err := e1.LastInsertId()
		if err != nil {
			return err
		}

		// insert image if there is one
		_, err = tx.Exec("INSERT INTO images (post_id,image_file,image_thumbnail,image_hash,image_orig_height,image_orig_width,image_tn_height,image_tn_width) VALUES (?,?,?,?,?,?,?,?)",
			p_id, i.Filename, i.Thumbnail, i.MD5, i.OrigHeight, i.OrigWidth, i.ThumbHeight, i.ThumbWidth)
		if err != nil {
			return err
		}

	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return
	}

	return

}
