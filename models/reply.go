package models

import (
	"database/sql"
	"errors"
	"html"

	"github.com/microcosm-cc/bluemonday"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// ReplyModel holds the request input
type ReplyModel struct {
	UID         uint
	Ib          uint
	Thread      uint
	IP          string
	Comment     string
	Filename    string
	Thumbnail   string
	MD5         string
	SHA         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
	Image       bool
}

// IsValid will check struct validity
func (m *ReplyModel) IsValid() bool {

	if m.UID == 0 {
		return false
	}

	if m.Ib == 0 {
		return false
	}

	if m.Thread == 0 {
		return false
	}

	if m.IP == "" {
		return false
	}

	if !m.Image && m.Comment == "" {
		return false
	}

	if m.Image {

		if m.Filename == "" {
			return false
		}

		if m.Thumbnail == "" {
			return false
		}

		if m.MD5 == "" {
			return false
		}

		if m.OrigWidth == 0 {
			return false
		}

		if m.OrigHeight == 0 {
			return false
		}

		if m.ThumbWidth == 0 {
			return false
		}

		if m.ThumbHeight == 0 {
			return false
		}

	}

	return true

}

// ValidateInput will make sure all the parameters are valid
func (m *ReplyModel) ValidateInput() (err error) {

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize for html and xss
	m.Comment = html.UnescapeString(p.Sanitize(m.Comment))

	// There must either be a comment, an image, or an image with a comment
	// If theres no image a comment is required
	comment := validate.Validate{Input: m.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}

	// if there is no image check the comment
	if !m.Image {
		if comment.IsEmpty() {
			return e.ErrNoComment
		} else if comment.MinLength() {
			return e.ErrCommentShort
		} else if comment.MaxLength() {
			return e.ErrCommentLong
		}
	}

	// If theres an image and a comment validate comment
	if m.Image && !comment.IsEmpty() {
		if comment.MinLength() {
			return e.ErrCommentShort
		} else if comment.MaxLength() {
			return e.ErrCommentLong
		}
	}

	return

}

// Status will return info about the thread
func (m *ReplyModel) Status() (err error) {
	// Get transaction handle with proper isolation level
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var closed bool
	var total uint

	// Lock the thread row for reading and potential update
	// Using FOR UPDATE to prevent other transactions from modifying this thread
	err = tx.QueryRow(`SELECT ib_id, thread_closed, count(post_num) FROM threads
    INNER JOIN posts on threads.thread_id = posts.thread_id
    WHERE threads.thread_id = ? AND post_deleted != 1
    FOR UPDATE`, m.Thread).Scan(&m.Ib, &closed, &total)
	if err != nil {
		// Check specifically for no rows (thread doesn't exist)
		if err == sql.ErrNoRows {
			return e.ErrNotFound
		}
		return
	}

	// Error if thread is closed
	if closed {
		// No need to commit as we're just reading
		return e.ErrThreadClosed
	}

	// Close thread if above max posts
	if total >= config.Settings.Limits.PostsMax {
		_, err = tx.Exec("UPDATE threads SET thread_closed=1 WHERE thread_id = ?",
			m.Thread)
		if err != nil {
			return err
		}

		// Commit the transaction to persist the thread closed status
		err = tx.Commit()
		if err != nil {
			return err
		}

		return e.ErrThreadClosed
	}

	// Commit the transaction to release the lock
	err = tx.Commit()
	if err != nil {
		return err
	}

	return
}

// Post will add the reply to the database with a transaction
func (m *ReplyModel) Post() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("ReplyModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// First get next post_num with row locking to prevent race conditions
	var nextPostNum uint
	err = tx.QueryRow(`SELECT COALESCE(MAX(post_num), 0) + 1 
                      FROM posts 
                      WHERE thread_id = ? 
                      FOR UPDATE`, m.Thread).Scan(&nextPostNum)
	if err != nil {
		return
	}

	// insert new post with the safely obtained post_num
	e1, err := tx.Exec(`INSERT INTO posts (thread_id, user_id, post_num, post_time, post_ip, post_text)
                      VALUES (?, ?, ?, NOW(), ?, ?)`,
		m.Thread, m.UID, nextPostNum, m.IP, m.Comment)
	if err != nil {
		return
	}

	if m.Image {
		var pID int64

		pID, err = e1.LastInsertId()
		if err != nil {
			return err
		}

		// insert image if there is one
		_, err = tx.Exec("INSERT INTO images (post_id,image_file,image_thumbnail,image_hash,image_sha,image_orig_height,image_orig_width,image_tn_height,image_tn_width) VALUES (?,?,?,?,?,?,?,?,?)",
			pID, m.Filename, m.Thumbnail, m.MD5, m.SHA, m.OrigHeight, m.OrigWidth, m.ThumbHeight, m.ThumbWidth)
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
