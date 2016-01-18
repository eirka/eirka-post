package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
)

// New tag input
type newTagForm struct {
	Ib   uint   `json:"ib" binding:"required"`
	Tag  string `json:"name" binding:"required"`
	Type uint   `json:"type" binding:"required"`
}

// NewTagController handles the creation of new tags
func NewTagController(c *gin.Context) {
	var err error
	var ntf newTagForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&ntf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("NewTagController.Bind")
		return
	}

	// Set parameters to NewTagModel
	m := models.NewTagModel{
		Ib:      ntf.Ib,
		Tag:     ntf.Tag,
		TagType: ntf.Type,
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("NewTagController.ValidateInput")
		return
	}

	// Check tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("NewTagController.Status")
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("NewTagController.Status")
		return
	}

	// Post data
	err = m.Post()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("NewTagController.Post")
		return
	}

	// Initialize cache handle
	cache := redis.RedisCache

	// Delete redis stuff
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)

	err = cache.Delete(tags_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("NewTagController.cache.Delete")
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

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("NewTagController.audit.Submit")
	}

	return

}
