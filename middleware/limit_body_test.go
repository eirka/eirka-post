package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/config"
)

// TestLimitBodyContentLength verifies the fast path: a request declaring an
// oversized Content-Length is rejected before the handler runs.
func TestLimitBodyContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Settings.Limits.ImageMaxSize = 1000

	r := gin.New()
	called := false
	r.POST("/upload", LimitBody(), func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	big := make([]byte, 1000+multipartOverhead+1)
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(big))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code, "oversized body should be rejected")
	assert.False(t, called, "handler should not run for an oversized body")
}

// TestLimitBodyUnderLimitPasses verifies an under-limit request reaches the handler.
func TestLimitBodyUnderLimitPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Settings.Limits.ImageMaxSize = 1000000

	r := gin.New()
	called := false
	r.POST("/upload", LimitBody(), func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("small body"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "under-limit body should pass")
	assert.True(t, called, "handler should run for an under-limit body")
}

// TestLimitBodyCapsStreamingBody verifies the MaxBytesReader backstop: a body
// with unknown length that streams past the cap errors when read, rather than
// being buffered without limit.
func TestLimitBodyCapsStreamingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.Settings.Limits.ImageMaxSize = 1000

	r := gin.New()
	var readErr error
	r.POST("/upload", LimitBody(), func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		c.Status(http.StatusOK)
	})

	big := bytes.Repeat([]byte("a"), 1000+multipartOverhead+100)
	req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(big))
	req.ContentLength = -1 // unknown length: skip the fast path, exercise the reader cap
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Error(t, readErr, "reading past the cap should error via MaxBytesReader")
}
