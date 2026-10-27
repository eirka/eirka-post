package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	u "github.com/eirka/eirka-post/utils"
)

// errInvalidLogin is returned for both an unknown name and a wrong password, so
// the response does not reveal which names are registered
var errInvalidLogin = errors.New("invalid name or password")

// dummyHash is compared against when the name has no account, so a login for an
// unknown name does the same bcrypt work as a wrong password, and its timing
// does not give the name away. It uses the same cost as user.HashPassword.
var dummyHash = sync.OnceValue(func() []byte {
	// only fails for an out of range cost or a password over 72 bytes
	hash, _ := bcrypt.GenerateFromPassword([]byte("not a real password"), bcrypt.DefaultCost)
	return hash
})

// Input from login form
type loginForm struct {
	Ib       uint   `json:"ib" binding:"required"`
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
		c.Error(err).SetMeta("LoginController.Bind")
		return
	}

	// check if the username is valid
	if !user.IsValidName(lf.Name) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrUserNotAllowed.Error()})
		c.Error(e.ErrUserNotAllowed).SetMeta("LoginController.user.IsValidName")
		return
	}

	// rate limit login by ip and by name, before the name is looked up, so
	// names without an account are limited too
	err = u.LoginCounter(lf.Name, c.ClientIP())
	if err == e.ErrMaxLogins {
		c.JSON(http.StatusTooManyRequests, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("LoginController.LoginCounter")
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("LoginController.LoginCounter")
		return
	}

	// default user
	usr := user.DefaultUser()

	// get user id from name, this populates the password hash
	err = usr.FromName(lf.Name)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, e.ErrUserNotValid) {
		// do the bcrypt work a wrong password would, and give the same answer
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(lf.Password))
		c.JSON(http.StatusBadRequest, gin.H{"error_message": errInvalidLogin.Error()})
		c.Error(err).SetMeta("LoginController.user.FromName")
		return
	} else if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("LoginController.user.FromName")
		return
	}

	// compare passwords
	if !usr.ComparePassword(lf.Password) {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": errInvalidLogin.Error()})
		c.Error(e.ErrInvalidPassword).SetMeta("LoginController.user.ComparePassword")
		return
	}

	// create jwt token
	token, err := usr.CreateToken()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("LoginController.user.CreateToken")
		return
	}

	// set the jwt cookie
	http.SetCookie(c.Writer, user.CreateCookie(token))

	c.JSON(http.StatusOK, gin.H{"success_message": "Login successful"})
}
