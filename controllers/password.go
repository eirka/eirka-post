package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
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
	userdata := c.MustGet("userdata").(u.User)

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

	// hash old password
	m.OldHashed, err = bcrypt.GenerateFromPassword([]byte(m.OldPw), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Compare old password in db to provided
	err = m.CheckOldPassword()
	if err == e.ErrInvalidPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}
	if err != nil {
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

	c.JSON(http.StatusOK, gin.H{"success_message": u.AuditChangePassword})

	audit := u.Audit{
		User:   userdata.Id,
		Ib:     pf.Ib,
		Ip:     c.ClientIP(),
		Action: u.AuditChangePassword,
		Info:   userdata.Name,
	}

	err = audit.Submit()
	if err != nil {
		c.Error(err)
	}

	return

}
