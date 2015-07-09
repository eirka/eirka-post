package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"

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

	// log user in
	err = m.Login()
	// invalid username, invalid password, user not confirmed, user locked, user banned
	if err == e.ErrInvalidUser || err == e.ErrInvalidPassword || err == e.ErrUserNotConfirmed || err == e.ErrUserLocked || err == e.ErrUserBanned {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	//
	// create hmac cookie
	//
	easyhmac.Secret = config.Settings.Session.Secret

	// Initialize SignedMessage struct with secret
	key := easyhmac.SignedMessage{}

	// Add payload data
	key.Payload = []byte(m.Sid)

	// Create HMAC signature
	key.Sign()

	// Marshal message to JSON and encode in url-safe base64
	signedkey, err := key.Encode()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	//
	// create hmac cookie
	//

	// make session cookie
	cookie := &http.Cookie{
		Name:     config.Settings.Session.CookieName,
		Value:    signedkey,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Domain:   ".trish.io",
		Path:     "/",
		HttpOnly: true,
	}

	// set cookie
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful"})

	return

}
