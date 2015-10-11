package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
)

// CloseController will toggle a threads Close bool
func CloseController(c *gin.Context) {

	// Get parameters from validate middleware
	params := c.MustGet("params").([]uint)

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	// Initialize model struct
	m := &models.CloseModel{
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

	// toggle status
	err = m.Toggle()
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
	thread_key := fmt.Sprintf("%s:%d:%d", "thread", m.Ib, m.Id)

	err = cache.Delete(index_key, directory_key, thread_key)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	var success_message string

	if m.Closed {
		success_message = u.AuditOpenThread
	} else {
		success_message = u.AuditCloseThread
	}

	// response message
	c.JSON(http.StatusOK, gin.H{"success_message": success_message})

	// audit log
	audit := u.Audit{
		User:   userdata.Id,
		Ib:     m.Ib,
		Ip:     c.ClientIP(),
		Action: success_message,
		Info:   fmt.Sprintf("%s", m.Name),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
