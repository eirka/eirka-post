package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs user agent for successful POST requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Continue to handler
		c.Next()

		// Only log POST requests
		if c.Request.Method != "POST" {
			return
		}

		// Only log successful requests (2xx and 3xx status codes)
		status := c.Writer.Status()
		if status < 200 || status >= 400 {
			return
		}

		// Get user agent from request headers
		userAgent := c.Request.UserAgent()
		if userAgent == "" {
			userAgent = "Unknown"
		}

		// Log the successful POST request with user agent
		log.Printf("[POST SUCCESS] %s %s | Status: %d | IP: %s | User-Agent: %s",
			c.Request.Method,
			c.Request.URL.Path,
			status,
			c.ClientIP(),
			userAgent,
		)
	}
}
