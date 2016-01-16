package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
)

// Input from new thread form
type registerForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

// RegisterController handles initial registration
func RegisterController(c *gin.Context) {
	var err error
	var rf registerForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&rf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
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

	// check if the username is valid
	if !user.IsValidName(m.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserNotAllowed.Error()})
		c.Error(e.ErrUserNotAllowed)
		return
	}

	// Check database for duplicate name
	if user.CheckDuplicate(m.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrDuplicateName.Error()})
		c.Error(e.ErrDuplicateName)
		return
	}

	// hash password
	m.Hashed, err = user.HashPassword(m.Password)
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
