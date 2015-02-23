package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
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
	var ntf newTagForm

	c.Bind(&ntf)

	// Set parameters to NewTagModel
	m := models.NewTagModel{
		Ip:      c.ClientIP(),
		Ib:      ntf.Ib,
		Tag:     ntf.Tag,
		TagType: ntf.Type,
	}

	// Test for antispam key from Prim
	antispam := ntf.Antispam
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(400, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey, "Operation aborted")
		return
	}

	// Validate input parameters
	err := m.ValidateInput()
	if err != nil {
		c.JSON(400, gin.H{"error_message": err.Error()})
		c.Error(err, "Operation aborted")
		return
	}

	// Check tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag {
		c.JSON(400, gin.H{"error_message": err.Error()})
		c.Error(err, "Operation aborted")
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err, "Operation aborted")
		return
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
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)

	err = cache.Delete(tags_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err, "Operation aborted")
		return
	}

	c.Redirect(303, fmt.Sprintf("//%s/", c.Request.URL.Host))

	audit := u.Audit{
		User:   1,
		Ib:     m.Ib,
		Ip:     m.Ip,
		Action: u.AuditNewTag,
		Info:   fmt.Sprintf("%s", m.Tag),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err, "Audit log")
	}

	return

}
