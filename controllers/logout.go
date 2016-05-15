package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/user"
)

// LogoutController deletes the user session
func LogoutController(c *gin.Context) {

	// unset the jwt cookie
	http.SetCookie(c.Writer, user.DeleteCookie())

	c.JSON(http.StatusOK, gin.H{"success_message": "Logout successful"})

	return

}
