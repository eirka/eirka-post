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
	Name    string `form:"name"`
	Title   string `form:"title" binding:"required"`
	Comment string `form:"comment" binding:"required"`
	Ib      uint   `form:"ib" binding:"required"`
}

// ThreadController handles the creation of new threads
func ThreadController(c *gin.Context) {
	var err error
	var tf threadForm
	req := c.Request

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
		Ip:      c.ClientIP(),
		Name:    tf.Name,
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
		Name:    m.Name,
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

	// Check image header ext
	err = image.CheckReqExt()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Get image MD5
	err = image.GetMD5()
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
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Check file for magic bytes and get ext
	err = image.CheckMagic()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Make filenames
	err = image.MakeFilenames()

	// Handle WebM
	if image.Ext == ".webm" {

		// Save the webm to a file
		err = image.SaveImage()
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err)
			return
		}

		// Get webm stats like size and dimensions
		err = image.CheckWebM()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err)
			return
		}

		// Make the thumbnail
		err = image.CreateWebMThumbnail()
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err)
			return
		}

		// Handle images
	} else {

		// Get image stats like size and dimensions
		err = image.GetStats()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
			c.Error(err)
			return
		}

		// Save the image to a file
		err = image.SaveImage()
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err)
			return
		}

		// Make the thumbnail
		err = image.CreateThumbnail()
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
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
		User:   1,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditNewThread,
		Info:   fmt.Sprintf("%d", m.Id),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
