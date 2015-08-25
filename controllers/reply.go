package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
)

// Input from new reply form
type replyForm struct {
	Key     string `form:"askey" binding:"required"`
	Comment string `form:"comment"`
	Thread  uint   `form:"thread" binding:"required"`
}

// ReplyController handles the creation of new threads
func ReplyController(c *gin.Context) {
	var err error
	var rf replyForm
	req := c.Request

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	err = c.Bind(&rf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Test for antispam key from Prim
	antispam := rf.Key
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey)
		return
	}

	// Set parameters to ReplyModel
	m := models.ReplyModel{
		Uid:     userdata.Id,
		Ip:      c.ClientIP(),
		Comment: rf.Comment,
		Thread:  rf.Thread,
		Image:   true,
	}

	image := u.ImageType{}

	// Check if theres a file
	image.File, image.Header, err = req.FormFile("file")
	if err == http.ErrMissingFile {
		m.Image = false
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Check thread status
	err = m.Status()
	if err == e.ErrThreadClosed {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	if m.Comment != "" {

		// Check comment in SFS and Akismet
		check := u.CheckComment{
			Ip:      m.Ip,
			Name:    userdata.Name,
			Ua:      req.UserAgent(),
			Referer: req.Referer(),
			Comment: m.Comment,
		}

		err = check.Get()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err)
			return
		}

	}

	if m.Image {

		// process the uploaded file, this creates an md5
		err = image.ProcessFile()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err)
			return
		}

		// Set MD5 from results
		m.MD5 = image.MD5

		// Initialize check duplicate
		duplicate := u.CheckDuplicate{
			Ib:  m.Ib,
			MD5: m.MD5,
		}

		// Check database for duplicate image hashes
		err = duplicate.Get()
		if err == e.ErrDuplicateImage {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error(), "thread": duplicate.Thread, "post": duplicate.Post})
			c.Error(err)
			return
		} else if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err)
			return
		}

		if image.Ext == ".webm" {

			// Save the webm to a file
			err = image.SaveWebM()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
				c.Error(err)
				return
			}

		} else {

			// Save the image to a file
			err = image.SaveImage()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
				c.Error(err)
				return
			}

		}

		m.OrigWidth = image.OrigWidth
		m.OrigHeight = image.OrigHeight
		m.ThumbWidth = image.ThumbWidth
		m.ThumbHeight = image.ThumbHeight
		m.Filename = image.Filename
		m.Thumbnail = image.Thumbnail

	}

	// Post data
	err = m.Post()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Initialize cache handle
	cache := u.RedisCache

	// Delete redis stuff
	index_key := fmt.Sprintf("%s:%d", "index", m.Ib)
	directory_key := fmt.Sprintf("%s:%d", "directory", m.Ib)
	thread_key := fmt.Sprintf("%s:%d:%d", "thread", m.Ib, m.Thread)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = cache.Delete(index_key, directory_key, thread_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// generate redirect link
	redir := u.Redirect{
		Id:      m.Ib,
		Referer: req.Referer(),
	}

	err = redir.Link()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.Redirect(303, redir.Url)

	audit := u.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditReply,
		Info:   fmt.Sprintf("%d/%d", m.Thread, m.PostNum),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
