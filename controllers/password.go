package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
)

// Input from change password form
type passwordForm struct {
	Ib    uint   `json:"ib" binding:"required"`
	OldPw string `json:"oldpw" binding:"required"`
	NewPw string `json:"newpw" binding:"required"`
}

// PasswordController handles initial registration
func PasswordController(c *gin.Context) {
	var err error
	var pf passwordForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&pf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("PasswordController.Bind")
		return
	}

	// Set parameters to PasswordModel
	m := models.PasswordModel{
		Uid:   userdata.Id,
		OldPw: pf.OldPw,
		NewPw: pf.NewPw,
	}

	// Validate input
	err = m.Validate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("PasswordController.Validate")
		return
	}

	// get the password from the database
	err = user.Password()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("PasswordController.user.Password")
		return
	}

	// we now have the users name
	m.Name = user.Name

	// compare passwords
	if !user.ComparePassword(m.OldPw) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidPassword.Error()})
		c.Error(e.ErrInvalidPassword).SetMeta("PasswordController.user.ComparePassword")
		return
	}

	// hash password
	m.NewHashed, err = user.HashPassword(m.NewPw)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("PasswordController.user.HashPassword")
		return
	}

	// update password
	err = m.Update()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("PasswordController.Update")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditChangePassword})

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     pf.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditChangePassword,
		Info:   m.Name,
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("PasswordController.audit.Submit")
	}

	return

}
