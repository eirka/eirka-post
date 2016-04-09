package models

import (
	"errors"
	"html"

	"github.com/microcosm-cc/bluemonday"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/validate"
)

// ThreadModel holds the request input
type ThreadModel struct {
	UID         uint
	Ib          uint
	IP          string
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

// IsValid will check struct validity
func (m *ThreadModel) IsValid() bool {

	if m.UID == 0 {
		return false
	}

	if m.Ib == 0 {
		return false
	}

	if m.IP == "" {
		return false
	}

	if m.Title == "" {
		return false
	}

	if m.Comment == "" {
		return false
	}

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

	return true

}

// ValidateInput will make sure all the parameters are valid
func (m *ThreadModel) ValidateInput() (err error) {

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// sanitize html and xss
	m.Title = html.UnescapeString(p.Sanitize(m.Title))

	// Validate title input
	title := validate.Validate{Input: m.Title, Max: config.Settings.Limits.TitleMaxLength, Min: config.Settings.Limits.TitleMinLength}
	if title.IsEmpty() {
		return e.ErrNoTitle
	} else if title.MinLength() {
		return e.ErrTitleShort
	} else if title.MaxLength() {
		return e.ErrTitleLong
	}

	// sanitize html and xss
	m.Comment = html.UnescapeString(p.Sanitize(m.Comment))

	// Validate comment input
	comment := validate.Validate{Input: m.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}
	if comment.IsEmpty() {
		return e.ErrNoComment
	} else if comment.MinLength() {
		return e.ErrCommentShort
	} else if comment.MaxLength() {
		return e.ErrCommentLong
	}

	return

}

// Post will add the thread to the database with a transaction
func (m *ThreadModel) Post() (err error) {

	// check model validity
	if !m.IsValid() {
		return errors.New("ThreadModel is not valid")
	}

	// Get transaction handle
	tx, err := db.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// insert into threads table
	e1, err := tx.Exec("INSERT INTO threads (ib_id,thread_title) VALUES (?,?)",
		m.Ib, m.Title)
	if err != nil {
		return
	}

	// Get new thread id
	tID, err := e1.LastInsertId()
	if err != nil {
		return
	}

	// insert into posts table
	e2, err := tx.Exec("INSERT INTO posts (thread_id,user_id,post_time,post_ip,post_text) VALUES (?,?,NOW(),?,?)",
		tID, m.UID, m.IP, m.Comment)
	if err != nil {
		return
	}

	// Get new post id
	pID, err := e2.LastInsertId()
	if err != nil {
		return
	}

	// insert into images table
	_, err = tx.Exec("INSERT INTO images (post_id,image_file,image_thumbnail,image_hash,image_orig_height,image_orig_width,image_tn_height,image_tn_width) VALUES (?,?,?,?,?,?,?,?)",
		pID, m.Filename, m.Thumbnail, m.MD5, m.OrigHeight, m.OrigWidth, m.ThumbHeight, m.ThumbWidth)
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
