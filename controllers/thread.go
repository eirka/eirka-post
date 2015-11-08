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

// Input from new thread form
type threadForm struct {
	Key     string `form:"askey" binding:"required"`
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
	userdata := c.MustGet("userdata").(u.User)

	// check size of content
	if req.ContentLength > int64(config.Settings.Limits.ImageMaxSize) {
		c.JSON(http.StatusExpectationFailed, gin.H{"error_message": e.ErrImageSize.Error()})
		c.Error(e.ErrImageSize)
		return
	}

	// set max bytes reader
	req.Body = http.MaxBytesReader(c.Writer, req.Body, int64(config.Settings.Limits.ImageMaxSize))

	err = c.Bind(&tf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Test for antispam key from Prim
	antispam := tf.Key
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey)
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
		c.Error(e.ErrNoImage)
		return
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

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

	err = cache.Delete(index_key, directory_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.Redirect(303, req.Referer())

	audit := u.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditNewThread,
		Info:   fmt.Sprintf("%s", m.Title),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
