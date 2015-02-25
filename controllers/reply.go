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

// ReplyController handles the creation of new threads
func ReplyController(c *gin.Context) {
	var err error
	req := c.Request

	// Test for antispam key from Prim
	antispam := req.FormValue("askey")
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(400, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey, "Operation aborted")
		return
	}

	// Set parameters to ReplyModel
	m := models.ReplyModel{
		Ip:      c.ClientIP(),
		Name:    req.FormValue("name"),
		Comment: req.FormValue("comment"),
		Image:   true,
	}

	image := u.ImageType{}

	// Check if theres a file
	image.File, image.Header, err = req.FormFile("file")
	if err == http.ErrMissingFile {
		m.Image = false
	}

	// Validate input parameters
	err = m.ValidateInput(req.FormValue("thread"))
	if err != nil {
		c.JSON(400, gin.H{"error_message": err.Error()})
		c.Error(err, "Operation aborted")
		return
	}

	// Check thread status
	err = m.Status()
	if err == e.ErrThreadClosed {
		c.JSON(400, gin.H{"error_message": err.Error()})
		c.Error(err, "Operation aborted")
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err, "Operation aborted")
		return
	}

	if m.Comment != "" {

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
			c.JSON(400, gin.H{"error_message": err.Error()})
			c.Error(err, "Operation aborted")
			return
		}

	}

	if m.Image {

		// Check image header ext
		err = image.CheckReqExt()
		if err != nil {
			c.JSON(400, gin.H{"error_message": err.Error()})
			c.Error(err, "Operation aborted")
			return
		}

		// Get image MD5
		err = image.GetMD5()
		if err != nil {
			c.JSON(400, gin.H{"error_message": err.Error()})
			c.Error(err, "Operation aborted")
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
			c.JSON(400, gin.H{"error_message": err.Error(), "thread": duplicate.Thread, "post": duplicate.Post})
			c.Error(err, "Operation aborted")
			return
		}
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err, "Operation aborted")
			return
		}

		// Check file for magic bytes and get ext
		err = image.CheckMagic()
		if err != nil {
			c.JSON(400, gin.H{"error_message": err.Error()})
			c.Error(err, "Operation aborted")
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
				c.Error(err, "Operation aborted")
				return
			}

			// Get webm stats like size and dimensions
			err = image.CheckWebM()
			if err != nil {
				c.JSON(400, gin.H{"error_message": err.Error()})
				c.Error(err, "Operation aborted")
				return
			}

			// Make the thumbnail
			err = image.CreateWebMThumbnail()
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrInternalError))
				c.Error(err, "Operation aborted")
				return
			}

			// Handle images
		} else {

			// Get image stats like size and dimensions
			err = image.GetStats()
			if err != nil {
				c.JSON(400, gin.H{"error_message": err.Error()})
				c.Error(err, "Operation aborted")
				return
			}

			// Save the image to a file
			err = image.SaveImage()
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrInternalError))
				c.Error(err, "Operation aborted")
				return
			}

			// Save the image to gcs
			err = image.UploadGCS()
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrInternalError))
				c.Error(err, "Operation aborted")
				return
			}

			// Make the thumbnail
			err = image.CreateThumbnail()
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrInternalError))
				c.Error(err, "Operation aborted")
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
		c.Error(err, "Operation aborted")
		return
	}

	// Initialize cache handle
	cache := u.RedisCache

	// Delete redis stuff
	index_key := fmt.Sprintf("%s:%d", "index", m.Ib)
	directory_key := fmt.Sprintf("%s:%d", "directory", m.Ib)
	thread_key := fmt.Sprintf("%s:%d", "thread", m.Thread)

	err = cache.Delete(index_key, directory_key, thread_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err, "Operation aborted")
		return
	}

	c.Redirect(303, u.RedirectLink(req.Referer()))

	audit := u.Audit{
		User:   1,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditReply,
		Info:   fmt.Sprintf("%d/%d", m.Thread, m.PostNum),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err, "Audit log")
	}

	return

}
