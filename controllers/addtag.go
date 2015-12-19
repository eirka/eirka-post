package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/auth"
	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"

	"github.com/eirka/eirka-post/models"
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
	var err error
	var atf addTagForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(auth.User)

	err = c.Bind(&atf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Set parameters to AddTagModel
	m := models.AddTagModel{
		Ib:    atf.Ib,
		Tag:   atf.Tag,
		Image: atf.Image,
	}

	// Test for antispam key from Prim
	antispam := atf.Antispam
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey)
		return
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Check image for correct ib and tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag || err == e.ErrNotFound {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	} else if err != nil {
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
	cache := redis.RedisCache

	// Delete redis stuff
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)
	tag_key := fmt.Sprintf("%s:%d:%d", "tag", m.Ib, m.Tag)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = cache.Delete(tags_key, tag_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditAddTag})

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditAddTag,
		Info:   fmt.Sprintf("%d", m.Image),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
