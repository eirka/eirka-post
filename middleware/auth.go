package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"

	"github.com/techjanitor/easyhmac"
	"github.com/techjanitor/pram-post/config"
	e "github.com/techjanitor/pram-post/errors"
	u "github.com/techjanitor/pram-post/utils"
)

// checks for session cookie and handles permissions
func Auth(perms Permissions) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error

		// set default anonymous user
		user := u.User{
			Id:    1,
			Group: 0,
		}

		// get session cookie
		sessioncookie, _ := c.Request.Cookie(config.Settings.Session.CookieName)

		// if there is a cookie we will get the value from it
		if sessioncookie != nil {

			// get session cookie data
			cookiepayload := sessioncookie.Value

			// set easyhmac secret from config
			easyhmac.Secret = config.Settings.Session.Secret

			// Initialize SignedMessage struct with secret
			message := easyhmac.SignedMessage{}

			// Decode message
			err = message.Decode(cookiepayload)
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrUnauthorized))
				c.Error(err)
				c.Abort()
				return
			}

			// Verify signature, returns a bool (true if verified)
			check := message.Verify()
			if !check {
				c.JSON(e.ErrorMessage(e.ErrUnauthorized))
				c.Error(err)
				c.Abort()
				return
			}

			// check the provided data
			uid, gid, err := u.ValidateSession(message.Payload)
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrUnauthorized))
				c.Error(err)
				c.Abort()
				return
			}

			// set user and group
			user.Id = uid
			user.Group = gid

		}

		fmt.Println(user.Id, user.Group)

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
