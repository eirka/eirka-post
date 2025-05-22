package middleware

import (
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
)

// Simple cache for banned IPs to reduce database load
// Key: IP address, Value: time when cache entry expires
var (
	bannedIPCache     = make(map[string]time.Time)
	bannedIPCacheLock sync.RWMutex
	cacheTTL          = 10 * time.Minute // Cache entries valid for 10 minutes
)

// isLocalhost checks if an IP is a localhost address
func isLocalhost(ip string) bool {
	// Check for common localhost string representations
	if ip == "localhost" {
		return true
	}

	// Parse the IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if it's a loopback address (127.0.0.0/8 or ::1)
	return parsedIP.IsLoopback()
}

// Bans will check if the client IP is in the banned list
func Bans() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Skip checks for empty IP (shouldn't happen with proper Gin setup)
		if clientIP == "" {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(errors.New("empty IP address")).SetMeta("Bans.EmptyIP")
			c.Abort()
			return
		}

		// Special handling for localhost IPs
		if isLocalhost(clientIP) {
			// Check if localhost is in the ban list
			isBanned, err := CheckBannedIP(clientIP)
			if err != nil {
				c.JSON(e.ErrorMessage(e.ErrInternalError))
				c.Error(err).SetMeta("Bans.CheckBannedIP")
				c.Abort()
				return
			}

			// If localhost is banned, log a warning but allow the request
			if isBanned {
				log.Printf("WARNING: Localhost IP (%s) is in the ban list. This indicates a proxy misconfiguration. Request allowed anyway.", clientIP)
			}

			// Continue with the request regardless of ban status for localhost
			c.Next()
			return
		}

		// For non-localhost IPs, proceed with normal ban checking
		isBanned, err := CheckBannedIP(clientIP)
		if err != nil {
			c.JSON(e.ErrorMessage(e.ErrInternalError))
			c.Error(err).SetMeta("Bans.CheckBannedIP")
			c.Abort()
			return
		}

		if isBanned {
			c.JSON(e.ErrorMessage(e.ErrForbidden))
			c.Error(e.ErrIPBanned).SetMeta("Bans.IPBanned")
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckBannedIP will check if the IP is in the banned list
// Uses a cache to reduce database load
func CheckBannedIP(ip string) (isBanned bool, err error) {
	// Validate IP address format
	if net.ParseIP(ip) == nil {
		return false, errors.New("invalid IP address format")
	}

	// Check cache first (read lock)
	bannedIPCacheLock.RLock()
	expiryTime, found := bannedIPCache[ip]
	bannedIPCacheLock.RUnlock()

	// If found in cache and not expired, return cached result
	if found && time.Now().Before(expiryTime) {
		return true, nil
	}

	// Not in cache or expired, check database
	dbase, err := db.GetDb()
	if err != nil {
		return false, err
	}

	var count int
	err = dbase.QueryRow(`SELECT count(*) FROM banned_ips WHERE ban_ip = ?`, ip).Scan(&count)
	if err != nil {
		return false, err
	}

	isBanned = count > 0

	// If banned, add to cache (write lock)
	if isBanned {
		bannedIPCacheLock.Lock()
		bannedIPCache[ip] = time.Now().Add(cacheTTL)
		bannedIPCacheLock.Unlock()
	}

	return isBanned, nil
}
