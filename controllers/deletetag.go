package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
)

// DeleteTagController will delete a tag
func DeleteTagController(c *gin.Context) {

	// Get parameters from validate middleware
	params := c.MustGet("params").([]uint)

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	// Initialize model struct
	m := &models.DeleteTagModel{
		Id: params[0],
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
	cache := u.RedisCache

	// Delete redis stuff
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)
	tag_key := fmt.Sprintf("%s:%d", "tag", m.Ib, m.Id)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)

	err = cache.Delete(tags_key, tag_key, image_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// response message
	c.JSON(http.StatusOK, gin.H{"success_message": u.AuditDeleteTag})

	// audit log
	audit := u.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: u.AuditDeleteTag,
		Info:   fmt.Sprintf("%d", m.Name),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
