package models

import (
	"github.com/microcosm-cc/bluemonday"
	"html"
	"net"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

type ReplyModel struct {
	Ib          uint
	Thread      uint
	PostNum     uint
	Ip          string
	Name        string
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

// ValidateInput will make sure all the parameters are valid
func (i *ReplyModel) ValidateInput() (err error) {
	if i.Thread == 0 {
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

	// Initialize bluemonday
	p := bluemonday.StrictPolicy()

	// Sanitize comment for html and xss
	i.Comment = p.Sanitize(i.Comment)

	i.Comment = html.UnescapeString(i.Comment)

	// There must either be a comment, an image, or an image with a comment
	// If theres no image a comment is required
	comment := u.Validate{Input: i.Comment, Max: config.Settings.Limits.CommentMaxLength, Min: config.Settings.Limits.CommentMinLength}

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
	db, err := u.GetDb()
	if err != nil {
		return
	}

	var closed bool
	var total, lastnum uint

	// Check if thread is closed and get the total amount of posts
	err = db.QueryRow(`SELECT ib_id,thread_closed,count(post_num),post_num 
	FROM ( SELECT ib_id,threads.thread_id,thread_closed,post_num 
	FROM threads  LEFT JOIN posts on threads.thread_id = posts.thread_id 
	WHERE threads.thread_id = ?
	GROUP BY post_num DESC) AS b`, i.Thread).Scan(&i.Ib, &closed, &total, &lastnum)
	if err != nil {
		return
	}

	// Error if thread is closed
	if closed {
		return e.ErrThreadClosed
	}

	// Close thread if above max posts
	if total > config.Settings.Limits.PostsMax {
		updatestatus, err := db.Prepare("UPDATE threads SET thread_closed=1 WHERE thread_id = ?")
		if err != nil {
			return err
		}
		defer updatestatus.Close()

		_, err = updatestatus.Exec(i.Thread)
		if err != nil {
			return err
		}

		return e.ErrThreadClosed
	}

	i.PostNum = lastnum + 1

	return

}

// Post will add the reply to the database with a transaction
func (i *ReplyModel) Post() (err error) {

	// Get transaction handle
	tx, err := u.GetTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// Insert data into posts table
	ps1, err := tx.Prepare("INSERT INTO posts (thread_id,post_name,post_num,post_time,post_ip,post_text) VALUES (?,?,?,NOW(),?,?)")
	if err != nil {
		return
	}
	defer ps1.Close()

	// Update thread last post time
	ps2, err := tx.Prepare("UPDATE threads SET thread_last_post = NOW() WHERE thread_id = ?")
	if err != nil {
		return
	}
	defer ps2.Close()

	e1, err := ps1.Exec(i.Thread, i.Name, i.PostNum, i.Ip, i.Comment)
	if err != nil {
		return
	}

	_, err = ps2.Exec(i.Thread)
	if err != nil {
		return
	}

	if i.Image {

		// Insert data into images table
		ps3, err := tx.Prepare("INSERT INTO images (post_id,image_file,image_thumbnail,image_hash,image_orig_height,image_orig_width,image_tn_height,image_tn_width) VALUES (?,?,?,?,?,?,?,?)")
		if err != nil {
			return err
		}
		defer ps2.Close()

		p_id, err := e1.LastInsertId()
		if err != nil {
			return err
		}

		_, err = ps3.Exec(p_id, i.Filename, i.Thumbnail, i.MD5, i.OrigHeight, i.OrigWidth, i.ThumbHeight, i.ThumbWidth)
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
