package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

// checks for session cookie and handles permissions
func Auth(perms Permissions) gin.HandlerFunc {
	return func(c *gin.Context) {

		// user data
		var user u.User

		// get session cookie
		cookie, err := c.Request.Cookie(config.Settings.Session.CookieName)
		if err == http.ErrNoCookie {
			// set as anonymous user if theres no cookie
			user = u.User{
				Id:    1,
				Group: 0,
			}

		}
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err)
			c.Abort()
			return
		}

		// get session cookie data

		// check if user meets set permissions
		if user.Group < perms.Minimum {
			c.JSON(e.ErrorMessage(e.ErrUnauthorized))
			c.Error(err)
			c.Abort()
			return
		}

		// set user data
		c.Set("userdata", user)

		c.Next()

	}
}

// permissions data
type Permissions struct {
	Minimum int
}

// All users
func All() Permissions {
	return Permissions{Minimum: 0}
}

// registered users
func Registered() Permissions {
	return Permissions{Minimum: 1}
}

// moderators
func Moderators() Permissions {
	return Permissions{Minimum: 2}
}

// admins
func Admins() Permissions {
	return Permissions{Minimum: 3}
}
