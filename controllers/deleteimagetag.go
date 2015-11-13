package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/techjanitor/pram-libs/audit"
	"github.com/techjanitor/pram-libs/auth"
	e "github.com/techjanitor/pram-libs/errors"
	"github.com/techjanitor/pram-libs/redis"

	"github.com/techjanitor/pram-post/models"
)

// DeleteImageTagController will delete an image tag
func DeleteImageTagController(c *gin.Context) {

	// Get parameters from validate middleware
	params := c.MustGet("params").([]uint)

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(auth.User)

	// Initialize model struct
	m := &models.DeleteImageTagModel{
		Image: params[0],
		Tag:   params[1],
	}

	// Check the record id and get further info
	err := m.Status()
	if err == e.ErrNotFound {
		c.JSON(e.ErrorMessage(e.ErrNotFound))
		c.Error(err)
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Delete data
	err = m.Delete()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Initialize cache handle
	cache := redis.RedisCache

	// Delete redis stuff
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)
	tag_key := fmt.Sprintf("%s:%d", "tag", m.Ib, m.Tag)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = cache.Delete(tags_key, tag_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// response message
	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditDeleteImageTag})

	// audit log
	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditDeleteImageTag,
		Info:   fmt.Sprintf("%d/%s", m.Image, m.Name),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
