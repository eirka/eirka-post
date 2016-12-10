package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/audit"
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

	err = c.Bind(&rf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("ReplyController.Bind")
		return
	}

	// Set parameters to ReplyModel
	m := models.ReplyModel{
		UID:     userdata.ID,
		IP:      c.ClientIP(),
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

	// replies dont need comments if they have an image
	if m.Comment != "" {

		// Check comment in SFS and Akismet
		akismet := u.Akismet{
			IP:      m.IP,
			Ua:      req.UserAgent(),
			Referer: req.Referer(),
			Comment: m.Comment,
		}

		err = akismet.Check()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err).SetMeta("ReplyController.CheckAkismet")
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
		m.SHA = image.SHA
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

	// needs a fake hash index
	err = redis.NewKey("index").SetKey(fmt.Sprintf("%d", m.Ib), "0").Delete()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.redis.Cache.Delete")
		return
	}

	directoryKey := fmt.Sprintf("%s:%d", "directory", m.Ib)
	threadKey := fmt.Sprintf("%s:%d:%d", "thread", m.Ib, m.Thread)
	imageKey := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = redis.Cache.Delete(directoryKey, threadKey, imageKey)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ReplyController.redis.Cache.Delete")
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
		User:   userdata.ID,
		Ib:     m.Ib,
		Type:   audit.BoardLog,
		IP:     m.IP,
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
