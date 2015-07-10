package controllers

import (
	"encoding/base64"
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
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

	// if account is not confirmed
	if !m.Confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserNotConfirmed.Error()})
		c.Error(e.ErrUserNotConfirmed)
		return
	}

	// if locked
	if m.Locked {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserLocked.Error()})
		c.Error(e.ErrUserLocked)
		return
	}

	// if banned
	if m.Banned {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserBanned.Error()})
		c.Error(e.ErrUserBanned)
		return
	}

	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)
	// Set some claims
	token.Claims["iss"] = "pram"
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	token.Claims["user_name"] = m.Name
	token.Claims["user_id"] = m.Id

	// base64 encode our session secret
	signingKey, err := base64.URLEncoding.DecodeString(config.Settings.Session.Secret)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	bearer := fmt.Sprintf("Bearer %s", tokenString)

	// set authorization header
	c.Writer.Header().Set("Authorization", bearer)

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful"})

	return

}
