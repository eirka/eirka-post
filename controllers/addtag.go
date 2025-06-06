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
)

// Add tag input on image page
type addTagForm struct {
	Ib    uint `json:"ib" binding:"required"`
	Tag   uint `json:"tag" binding:"required"`
	Image uint `json:"image" binding:"required"`
}

// AddTagController handles the addition of a tag to an image
func AddTagController(c *gin.Context) {
	var err error
	var atf addTagForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&atf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("AddTagController.Bind")
		return
	}

	// Set parameters to AddTagModel
	m := models.AddTagModel{
		Ib:    atf.Ib,
		Tag:   atf.Tag,
		Image: atf.Image,
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("AddTagController.ValidateInput")
		return
	}

	// Check image for correct ib and tag for duplicate
	err = m.Status()
	if err == e.ErrDuplicateTag || err == e.ErrNotFound {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("AddTagController.Status")
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("AddTagController.Status")
		return
	}

	// Post data
	err = m.Post()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("AddTagController.Post")
		return
	}

	// Delete redis stuff
	tagsKey := fmt.Sprintf("%s:%d", "tags", m.Ib)
	tagKey := fmt.Sprintf("%s:%d:%d", "tag", m.Ib, m.Tag)
	imageKey := fmt.Sprintf("%s:%d", "image", m.Ib)

	// Continue even if redis fails since tag was already added successfully
	redisErr := redis.Cache.Delete(tagsKey, tagKey, imageKey)
	if redisErr != nil {
		c.Error(redisErr).SetMeta("AddTagController.redis.Cache.Delete")
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditAddTag})

	audit := audit.Audit{
		User:   userdata.ID,
		Ib:     m.Ib,
		Type:   audit.BoardLog,
		IP:     c.ClientIP(),
		Action: audit.AuditAddTag,
		Info:   fmt.Sprintf("%d", m.Image),
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("AddTagController.audit.Submit")
	}

}
