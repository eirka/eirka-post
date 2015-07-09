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

// Input from new thread form
type registerForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Key      string `json:"askey" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterController handles initial registration
func RegisterController(c *gin.Context) {
	var err error
	var rf registerForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	err = c.Bind(&rf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Test for antispam key from Prim
	antispam := rf.Key
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey)
		return
	}

	// Set parameters to RegisterModel
	m := models.RegisterModel{
		Name:     rf.Name,
		Email:    rf.Email,
		Password: rf.Password,
	}

	// Validate input
	err = m.Validate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Check database for duplicate name
	err = m.CheckDuplicate()
	if err == e.ErrDuplicateName {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// register user
	err = m.Register()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": u.AuditRegister})

	audit := u.Audit{
		User:   userdata.Id,
		Ib:     rf.Ib,
		Ip:     c.ClientIP(),
		Action: u.AuditRegister,
		Info:   fmt.Sprintf("%d", m.Name),
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
