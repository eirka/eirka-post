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

// Input from new thread form
type threadForm struct {
	Title   string `form:"title" binding:"required"`
	Comment string `form:"comment" binding:"required"`
	Ib      uint   `form:"ib" binding:"required"`
}

// ThreadController handles the creation of new threads
func ThreadController(c *gin.Context) {
	var err error
	var tf threadForm
	req := c.Request

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&tf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("ThreadController.Bind")
		return
	}

	// Set parameters to ThreadModel
	m := models.ThreadModel{
		UID:     userdata.ID,
		IP:      c.ClientIP(),
		Title:   tf.Title,
		Comment: tf.Comment,
		Ib:      tf.Ib,
	}

	image := u.ImageType{}

	// Check if theres a file
	image.File, image.Header, err = req.FormFile("file")
	if err == http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrNoImage.Error()})
		c.Error(e.ErrNoImage).SetMeta("ThreadController.FormFile")
		return
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("ThreadController.ValidateInput")
		return
	}

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
		c.Error(err).SetMeta("ThreadController.CheckAkismet")
		return
	}

	// set the ib for duplicate checking
	image.Ib = m.Ib

	// Save the image to a file
	err = image.SaveImage()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("ThreadController.SaveImage")
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

	// Post data
	err = m.Post()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ThreadController.Post")
		return
	}

	// needs a fake hash index
	// Continue even if redis fails since thread was already added successfully
	redisErr := redis.NewKey("index").SetKey(fmt.Sprintf("%d", m.Ib), "0").Delete()
	if redisErr != nil {
		c.Error(redisErr).SetMeta("ThreadController.redis.Index.Delete")
	}

	directoryKey := fmt.Sprintf("%s:%d", "directory", m.Ib)

	// Continue even if redis fails since thread was already added successfully
	redisErr = redis.Cache.Delete(directoryKey)
	if redisErr != nil {
		c.Error(redisErr).SetMeta("ThreadController.redis.Cache.Delete")
	}

	// get board domain and redirect to it
	redirect, err := u.Link(m.Ib, req.Referer())
	if err != nil {
		// Non-critical error, we can still redirect to the referer
		c.Error(err).SetMeta("ThreadController.redirect")
		redirect = req.Referer()
	}

	c.Redirect(303, redirect)

	audit := audit.Audit{
		User:   userdata.ID,
		Ib:     m.Ib,
		Type:   audit.BoardLog,
		IP:     m.IP,
		Action: audit.AuditNewThread,
		Info:   m.Title,
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("ThreadController.audit.Submit")
	}

}
