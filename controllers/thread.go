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

	// check size of content
	if req.ContentLength > int64(config.Settings.Limits.ImageMaxSize) {
		c.JSON(http.StatusExpectationFailed, gin.H{"error_message": e.ErrImageSize.Error()})
		c.Error(e.ErrImageSize).SetMeta("ThreadController.ContentLength")
		return
	}

	// set max bytes reader
	req.Body = http.MaxBytesReader(c.Writer, req.Body, int64(config.Settings.Limits.ImageMaxSize))

	err = c.Bind(&tf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("ThreadController.Bind")
		return
	}

	// Set parameters to ThreadModel
	m := models.ThreadModel{
		Uid:     userdata.Id,
		Ip:      c.ClientIP(),
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
		Ip:      m.Ip,
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

	// Initialize cache handle
	cache := redis.RedisCache

	// Delete redis stuff
	err = redis.NewKey("index").SetKey(fmt.Sprintf("%d", m.Ib)).Delete()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ThreadController.cache.Delete")
		return
	}

	directory_key := fmt.Sprintf("%s:%d", "directory", m.Ib)

	err = cache.Delete(directory_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ThreadController.cache.Delete")
		return
	}

	// get board domain and redirect to it
	redirect, err := u.Link(m.Ib, req.Referer())
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("ThreadController.redirect")
		return
	}

	c.Redirect(303, redirect)

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: audit.AuditNewThread,
		Info:   fmt.Sprintf("%s", m.Title),
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("ThreadController.audit.Submit")
	}

	return

}
