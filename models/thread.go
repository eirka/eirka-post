package models

import (
	"github.com/microcosm-cc/bluemonday"
	"html"
	"net"

	"pram-post/config"
	e "pram-post/errors"
	u "pram-post/utils"
)

type ThreadModel struct {
	Ib          uint
	Ip          string
	Name        string
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

// ValidateInput will make sure all the parameters are valid
func (i *ThreadModel) ValidateInput(ib string) (err error) {
	// Validate ib parameter
	i.Ib, err = u.ValidateParam(ib)
	if err != nil {
		return e.ErrInvalidParam
	}

	if i.Ib == 0 {
		return e.ErrInvalidParam
	}

	// Get the ip without port
	i.Ip, _, err = net.SplitHostPort(i.Ip)
	if err != nil {
		return e.ErrIpParse
	}

	// Make sure IP can be parsed
	if u.ValidateIP(i.Ip) {
		return e.ErrIpParse
	}

	// Validate name input
	name := u.Validate{Input: i.Name, Max: config.Settings.Limits.NameMaxLength, Min: config.Settings.Limits.NameMinLength}
	if name.IsEmpty() {
		i.Name = config.Settings.General.DefaultName
	} else if name.MinLength() {
		return e.ErrNameShort
	} else if name.MaxLength() {
		return e.ErrNameLong
	}

	// Validate title input
	title := u.Validate{Input: i.Title, Max: config.Settings.Limits.TitleMaxLength, Min: config.Settings.Limits.TitleMinLength}
	if title.IsEmpty() {
		return e.ErrNoTitle
	} else if title.MinLength() {
		return e.ErrTitleShort
	} else if title.MaxLength() {
		return e.ErrTitleLong
	}

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// Sanitize comment for html and xss
	i.Comment = p.Sanitize(i.Comment)

	i.Comment = html.UnescapeString(i.Comment)

	// Validate comment input
	comment := u.Validate{Input: i.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}
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
func (i *ThreadModel) Post() (err error) {

	// Get transaction handle
	tx, err := u.GetTransaction()
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
	ps2, err := tx.Prepare("INSERT INTO posts (thread_id,post_name,post_time,post_ip,post_text) VALUES (?,?,NOW(),?,?)")
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

	e2, err := ps2.Exec(t_id, i.Name, i.Ip, i.Comment)
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
