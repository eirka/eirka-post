package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// Bans will check ip with the banned list
func Bans() gin.HandlerFunc {
	return func(c *gin.Context) {

		// check ip against stop forum spam
		check, err := CheckBannedIps(c.ClientIP())
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err).SetMeta("Bans.CheckBannedIps")
			c.Abort()
			return
		}

		if check {
			c.JSON(e.ErrorMessage(e.ErrUnauthorized))
			c.Error(e.ErrIpBanned).SetMeta("Bans.CheckBannedIps")
			c.Abort()
			return
		}

		c.Next()

	}
}

// CheckBannedIps will check the database for the ip
func CheckBannedIps(ip string) (check bool, err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	err = dbase.QueryRow(`SELECT count(*) FROM banned_ips WHERE ban_ip = ?`, ip).Scan(&check)
	if err != nil {
		return
	}

	return
}
