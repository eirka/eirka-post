package controllers

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	u "github.com/eirka/eirka-post/utils"
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

	// default user
	user := user.DefaultUser()

	// get user id from name, this populates the password hash
	err = user.FromName(lf.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserNotExist.Error()})
		c.Error(err)
		return
	}

	// rate limit login
	err = u.LoginCounter(user.Id)
	if err != nil {
		c.JSON(429, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// compare passwords
	if !user.ComparePassword(lf.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidPassword.Error()})
		c.Error(e.ErrInvalidPassword)
		return
	}

	// create jwt token
	token, err := user.CreateToken()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful", "token": token})

	return

}
