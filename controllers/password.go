package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.com/techjanitor/pram-libs/audit"
	"github.com/techjanitor/pram-libs/auth"
	e "github.com/techjanitor/pram-libs/errors"

	"github.com/techjanitor/pram-post/models"
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
	userdata := c.MustGet("userdata").(auth.User)

	err = c.Bind(&pf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
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
		c.Error(err)
		return
	}

	// Get old password in db to provided
	err = m.GetOldPassword()
	if err == e.ErrInvalidPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// compare provided password to stored hash
	err = bcrypt.CompareHashAndPassword(m.OldHashed, []byte(m.OldPw))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidPassword.Error()})
		c.Error(err)
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// hash new password
	m.NewHashed, err = bcrypt.GenerateFromPassword([]byte(m.NewPw), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// update password
	err = m.Update()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditChangePassword})

	audit := audit.Audit{
		User:   userdata.Id,
		Ib:     pf.Ib,
		Ip:     c.ClientIP(),
		Action: audit.AuditChangePassword,
		Info:   userdata.Id,
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
