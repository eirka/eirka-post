package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
)

// DeletePostController will delete a tag
func DeletePostController(c *gin.Context) {

	// Get parameters from validate middleware
	params := c.MustGet("params").([]uint)

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	// Initialize model struct
	m := &models.DeletePostModel{
		Thread: params[0],
		Id:     params[1],
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
	index_key := fmt.Sprintf("%s:%d", "index", m.Ib)
	directory_key := fmt.Sprintf("%s:%d", "directory", m.Ib)
	thread_key := fmt.Sprintf("%s:%d:%d", "thread", m.Ib, m.Thread)
	post_key := fmt.Sprintf("%s:%d:%d", "post", m.Ib, m.Thread)
	tags_key := fmt.Sprintf("%s:%d", "tags", m.Ib)
	image_key := fmt.Sprintf("%s:%d", "image", m.Ib)
	new_key := fmt.Sprintf("%s:%d", "new", m.Ib)
	popular_key := fmt.Sprintf("%s:%d", "popular", m.Ib)

	err = cache.Delete(index_key, directory_key, thread_key, post_key, tags_key, image_key, new_key, popular_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// response message
	c.JSON(http.StatusOK, gin.H{"success_message": u.AuditDeletePost})

	// audit log
	audit := u.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: u.AuditDeletePost,
		Info:   fmt.Sprintf("%s/%d", m.Name, m.Id),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
