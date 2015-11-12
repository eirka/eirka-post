package controllers

import (
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"

	"github.com/techjanitor/pram-libs/auth"
	"github.com/techjanitor/pram-libs/config"
	e "github.com/techjanitor/pram-libs/errors"

	local "github.com/techjanitor/pram-post/config"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
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
		c.JSON(429, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// compare provided password to stored hash
	err = bcrypt.CompareHashAndPassword(m.Hash, []byte(m.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrInvalidPassword.Error()})
		c.Error(err)
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// set uset struct
	user := auth.User{
		Id: m.Id,
	}

	// get the rest of the user info
	err = user.Info()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		c.Abort()
		return
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)
	// Set some claims
	token.Claims["iss"] = "pram"
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Hour * 24 * 90).Unix()
	token.Claims["user_id"] = m.Id

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString([]byte(local.Settings.Session.Secret))
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful", "token": tokenString})

	return

}
