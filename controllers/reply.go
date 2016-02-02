package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
	u "github.com/eirka/eirka-post/utils"
)

// Input from new reply form
type replyForm struct {
	Comment string `form:"comment"`
	Thread  uint   `form:"thread" binding:"required"`
}

// ReplyController handles the creation of new threads
func ReplyController(c *gin.Context) {
	var err error
	var rf replyForm
	req := c.Request

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	// check size of content
	if req.ContentLength > int64(config.Settings.Limits.ImageMaxSize) {
		c.JSON(http.StatusExpectationFailed, gin.H{"error_message": e.ErrImageSize.Error()})
		c.Error(e.ErrImageSize).SetMeta("ReplyController.ContentLength")
		return
	}

	// set max bytes reader
	req.Body = http.MaxBytesReader(c.Writer, req.Body, int64(config.Settings.Limits.ImageMaxSize))

	err = c.Bind(&rf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("ReplyController.Bind")
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
		c.Error(err).SetMeta("ReplyController.ValidateInput")
		return
	}

	// Check thread status
	err = m.Status()
	if err == e.ErrThreadClosed {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("ReplyController.Status")
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.Status")
		return
	}

	if m.Comment != "" {

		// Check comment in SFS and Akismet
		check := u.CheckComment{
			Ip:      m.Ip,
			Ua:      req.UserAgent(),
			Referer: req.Referer(),
			Comment: m.Comment,
		}

		err = check.Get()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err).SetMeta("ReplyController.CheckComment")
			return
		}

	}

	if m.Image {

		// set the ib for duplicate checking
		image.Ib = m.Ib

		// Save the image to a file
		err = image.SaveImage()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err).SetMeta("ReplyController.SaveImage")
			return
		}

		m.MD5 = image.MD5
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
		c.Error(err).SetMeta("ReplyController.Post")
		return
	}

	// Initialize cache handle
	cache := redis.RedisCache

	err = redis.NewKey("index").SetKey(fmt.Sprintf("%d", m.Ib)).Delete()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.cache.Delete")
		return
	}

	directory_key := fmt.Sprintf("%s:%d", "directory", m.Ib)
	thread_key := fmt.Sprintf("%s:%d:%d", "thread", m.Ib, m.Thread)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = cache.Delete(directory_key, thread_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.cache.Delete")
		return
	}

	// get board domain and redirect to it
	redirect, err := u.Link(m.Ib, req.Referer())
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.redirect")
		return
	}

	c.Redirect(303, redirect)

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: audit.AuditReply,
		Info:   fmt.Sprintf("%d", m.Thread),
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("ReplyController.audit.Submit")
	}

	return

}
