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

// Add tag input on image page
type addTagForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Tag      uint   `json:"tag" binding:"required"`
	Image    uint   `json:"image" binding:"required"`
	Antispam string `json:"askey" binding:"required"`
}

// AddTagController handles the creation of new threads
func AddTagController(c *gin.Context) {
	var atf addTagForm

	if !c.Bind(&atf) {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		return
	}

	// Set parameters to AddTagModel
	m := models.AddTagModel{
		Ip:    c.ClientIP(),
		Ib:    atf.Ib,
		Tag:   atf.Tag,
		Image: atf.Image,
	}

	// Test for antispam key from Prim
	antispam := atf.Antispam
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey, "Operation aborted")
		return
	}

	// Validate input parameters
	err := m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Check tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
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
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)
	tag_key := fmt.Sprintf("%s:%d", "tag", m.Tag)
	image_key := fmt.Sprintf("%s:%d", "image", m.Image)

	err = cache.Delete(tags_key, tag_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.Redirect(303, "/")

	audit := u.Audit{
		User:   1,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditAddTag,
		Info:   fmt.Sprintf("%d", m.Image),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
