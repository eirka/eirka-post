package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
)

// multipartOverhead is slack added on top of the image size limit to account for
// multipart boundaries, the comment/title form fields, and other form headers.
const multipartOverhead = 1 << 20 // 1 MiB

// LimitBody caps the size of an upload request body. It rejects a request whose
// declared Content-Length already exceeds the limit, and wraps the body with
// http.MaxBytesReader so an oversized streaming body is aborted while being read
// — before it is fully buffered into memory or spilled to a temp file on disk.
//
// This must run before anything that reads the body (SpamFilter's c.PostForm, or
// the controllers' FormFile/Bind), so it is registered first on upload routes.
// The limit is read per-request from config so it tracks the DB-backed setting.
func LimitBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		max := int64(config.Settings.Limits.ImageMaxSize) + multipartOverhead

		// Fast path: reject immediately when the client declares an oversized body.
		if c.Request.ContentLength > max {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error_message": e.ErrImageSize.Error()})
			c.Error(e.ErrImageSize).SetMeta("LimitBody.ContentLength")
			c.Abort()
			return
		}

		// Backstop for chunked/unknown-length bodies: cap the body as it streams.
		// A read past the limit returns an error to whatever parses the body.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)

		c.Next()
	}
}
