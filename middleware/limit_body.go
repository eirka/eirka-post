package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
)

const (
	// multipartOverhead is slack added on top of the image size limit to account
	// for multipart boundaries, the comment/title form fields, and other form headers.
	multipartOverhead = 1 << 20 // 1 MiB

	// maxBodySize caps every body that is not a multipart upload. The JSON and
	// urlencoded routes (login, register, tags, favorites, password, email) take
	// a handful of short fields.
	maxBodySize = 64 << 10 // 64 KiB
)

// LimitBody caps the size of every request body. It rejects a request whose
// declared Content-Length already exceeds the limit, and wraps the body with
// http.MaxBytesReader so an oversized streaming body is aborted while being read
// — before it is fully buffered into memory or spilled to a temp file on disk.
//
// It is registered globally, ahead of csrf.Verify: a request without the
// X-XSRF-TOKEN header has its token read from the form, and reading one form
// value parses the entire body. The upload forms post that way, so on the given
// upload routes a multipart body may be up to ImageMaxSize plus
// multipartOverhead. Every other body, including multipart sent to any other
// route, is capped at maxBodySize, so a route that binds a few fields never
// parses an upload-sized form. The upload limit is read per-request from config
// so it tracks the DB-backed setting.
func LimitBody(uploads ...string) gin.HandlerFunc {
	upload := make(map[string]bool, len(uploads))
	for _, path := range uploads {
		upload[path] = true
	}

	return func(c *gin.Context) {
		max := int64(maxBodySize)
		var tooLarge error = e.ErrInvalidParam
		// FullPath is the matched route pattern, and empty for NoRoute
		if upload[c.FullPath()] && c.ContentType() == binding.MIMEMultipartPOSTForm {
			max = int64(config.Settings.Limits.ImageMaxSize) + multipartOverhead
			tooLarge = e.ErrImageSize
		}

		// Fast path: reject immediately when the client declares an oversized body.
		if c.Request.ContentLength > max {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error_message": tooLarge.Error()})
			c.Error(tooLarge).SetMeta("LimitBody.ContentLength")
			c.Abort()
			return
		}

		// Backstop for chunked/unknown-length bodies: cap the body as it streams.
		// A read past the limit returns an error to whatever parses the body.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)

		c.Next()
	}
}
