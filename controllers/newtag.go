package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/techjanitor/pram-libs/audit"
	"github.com/techjanitor/pram-libs/auth"
	"github.com/techjanitor/pram-libs/config"
	e "github.com/techjanitor/pram-libs/errors"
	"github.com/techjanitor/pram-libs/redis"

	"github.com/techjanitor/pram-post/models"
)

// New tag input
type newTagForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Tag      string `json:"name" binding:"required"`
	Type     uint   `json:"type" binding:"required"`
	Antispam string `json:"askey" binding:"required"`
}

// NewTagController handles the creation of new threads
func NewTagController(c *gin.Context) {
	var err error
	var ntf newTagForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(auth.User)

	err = c.Bind(&ntf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Set parameters to NewTagModel
	m := models.NewTagModel{
		Ib:      ntf.Ib,
		Tag:     ntf.Tag,
		TagType: ntf.Type,
	}

	// Test for antispam key from Prim
	antispam := ntf.Antispam
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

	// Check tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag {
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

	err = cache.Delete(tags_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditNewTag})

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditNewTag,
		Info:   fmt.Sprintf("%s", m.Tag),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
