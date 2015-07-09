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
		_, err := c.Request.Cookie(config.Settings.Session.CookieName)
		if err == http.ErrNoCookie {
			// set as anonymous user if theres no cookie
			user = u.User{
				Id:    1,
				Group: 0,
			}

		}

		// get session cookie data

		// check if user meets set permissions
		if user.Group < perms.Minimum {
			c.JSON(e.ErrorMessage(e.ErrUnauthorized))
			c.Error(e.ErrUnauthorized)
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
	Minimum uint
}

func SetAuthLevel() Permissions {
	return Permissions{}
}

// All users
func (p Permissions) All() Permissions {
	p.Minimum = 0
	return p
}

// registered users
func (p Permissions) Registered() Permissions {
	p.Minimum = 1
	return p
}

// moderators
func (p Permissions) Moderators() Permissions {
	p.Minimum = 2
	return p
}

// admins
func (p Permissions) Admins() Permissions {
	p.Minimum = 3
	return p
}
