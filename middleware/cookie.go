package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
)

func GetAntiSpamCookie() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Test for cookie from Prim
		cookie, err := c.Request.Cookie(config.Settings.Antispam.CookieName)
		if err != nil {
			c.JSON(400, gin.H{"error_message": e.ErrNoCookie.Error()})
			c.Error(e.ErrNoCookie, "Operation aborted")
			c.Abort()
			return
		}

		// See if the cookie is the right value
		if cookie.Value != config.Settings.Antispam.CookieValue {
			c.JSON(400, gin.H{"error_message": e.ErrInvalidCookie.Error()})
			c.Error(e.ErrInvalidCookie, "Operation aborted")
			c.Abort()
			return
		}

		c.Next()

	}
}
