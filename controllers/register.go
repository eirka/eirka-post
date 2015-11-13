package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.com/techjanitor/pram-libs/audit"
	"github.com/techjanitor/pram-libs/auth"
	"github.com/techjanitor/pram-libs/config"
	e "github.com/techjanitor/pram-libs/errors"

	"github.com/techjanitor/pram-post/models"
)

// Input from new thread form
type registerForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Key      string `json:"askey" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// RegisterController handles initial registration
func RegisterController(c *gin.Context) {
	var err error
	var rf registerForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(auth.User)

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
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// hash password
	m.Hashed, err = bcrypt.GenerateFromPassword([]byte(m.Password), bcrypt.DefaultCost)
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

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditRegister})

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     rf.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditRegister,
		Info:   m.Name,
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
