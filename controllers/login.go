package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.com/techjanitor/easyhmac"
	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
)

// Input from login form
type loginForm struct {
	Ib       uint   `json:"ib" binding:"required"`
	Key      string `json:"askey" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginController handles user login
func LoginController(c *gin.Context) {
	var err error
	var lf loginForm

	err = c.Bind(&lf)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Test for antispam key from Prim
	antispam := lf.Key
	if antispam != config.Settings.Antispam.AntispamKey {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidKey.Error()})
		c.Error(e.ErrInvalidKey)
		return
	}

	// Set parameters to LoginModel
	m := models.LoginModel{
		Name:     lf.Name,
		Password: lf.Password,
	}

	// Validate input
	err = m.Validate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// get user info
	err = m.Query()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// rate limit login
	err = u.LoginCounter(m.Id)
	if err != nil {
		return
	}

	// compare provided password to stored hash
	err = bcrypt.CompareHashAndPassword(m.Hash, []byte(m.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return e.ErrInvalidPassword
	} else if err != nil {
		return e.ErrInternalError
	}

	// if account is not confirmed
	if !m.Confirmed {
		return e.ErrUserNotConfirmed
	}

	// if locked
	if m.Locked {
		return e.ErrUserLocked
	}

	// if banned
	if m.Banned {
		return e.ErrUserBanned
	}

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful"})

	return

}
