package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
)

func TestLimitBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := config.Settings.Limits.ImageMaxSize
	config.Settings.Limits.ImageMaxSize = 1000
	defer func() { config.Settings.Limits.ImageMaxSize = orig }()

	uploadMax := 1000 + multipartOverhead

	tests := []struct {
		name string
		// path is /upload, an upload route, unless set
		path        string
		contentType string
		size        int
		// chunked hides the length, so the fast path is skipped and only the
		// MaxBytesReader backstop applies
		chunked bool
		// wantCode is 0 when the handler runs; wantReadErr is whether its read
		// of the body hit the cap
		wantCode    int
		wantMessage string
		wantReadErr bool
	}{
		{name: "multipart under the upload limit", contentType: "multipart/form-data; boundary=x", size: uploadMax},
		{name: "multipart larger than the small limit", contentType: "multipart/form-data; boundary=x", size: maxBodySize * 4},
		{name: "multipart over the upload limit", contentType: "multipart/form-data; boundary=x", size: uploadMax + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrImageSize.Error()},
		{name: "chunked multipart over the upload limit", contentType: "multipart/form-data; boundary=x", size: uploadMax + 1, chunked: true, wantReadErr: true},
		{name: "json under the small limit", contentType: "application/json", size: maxBodySize},
		{name: "json over the small limit", contentType: "application/json", size: maxBodySize + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrInvalidParam.Error()},
		{name: "chunked json over the small limit", contentType: "application/json", size: maxBodySize + 1, chunked: true, wantReadErr: true},
		{name: "urlencoded over the small limit", contentType: "application/x-www-form-urlencoded", size: maxBodySize + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrInvalidParam.Error()},
		{name: "chunked urlencoded over the small limit", contentType: "application/x-www-form-urlencoded", size: maxBodySize + 1, chunked: true, wantReadErr: true},
		{name: "no content type over the small limit", size: maxBodySize + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrInvalidParam.Error()},
		{name: "multipart lookalike gets the small limit", contentType: "multipart/mixed; boundary=x", size: maxBodySize + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrInvalidParam.Error()},
		{name: "multipart to another route gets the small limit", path: "/other", contentType: "multipart/form-data; boundary=x", size: maxBodySize + 1, wantCode: http.StatusRequestEntityTooLarge, wantMessage: e.ErrInvalidParam.Error()},
		{name: "chunked multipart to another route", path: "/other", contentType: "multipart/form-data; boundary=x", size: maxBodySize + 1, chunked: true, wantReadErr: true},
		{name: "multipart to another route under the small limit", path: "/other", contentType: "multipart/form-data; boundary=x", size: maxBodySize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			var read int
			var readErr error
			handler := func(c *gin.Context) {
				called = true
				var body []byte
				body, readErr = io.ReadAll(c.Request.Body)
				read = len(body)
				c.Status(http.StatusOK)
			}

			r := gin.New()
			r.Use(LimitBody("/upload"))
			r.POST("/upload", handler)
			r.POST("/other", handler)

			path := tt.path
			if path == "" {
				path = "/upload"
			}
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(make([]byte, tt.size)))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.chunked {
				req.ContentLength = -1
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.wantCode != 0 {
				assert.Equal(t, tt.wantCode, w.Code)
				assert.Contains(t, w.Body.String(), tt.wantMessage)
				assert.False(t, called, "handler should not run for an oversized body")
				return
			}

			assert.True(t, called, "handler should run")
			if tt.wantReadErr {
				var maxErr *http.MaxBytesError
				assert.True(t, errors.As(readErr, &maxErr), "reading past the cap should fail with MaxBytesError, got %v", readErr)
				return
			}
			assert.NoError(t, readErr)
			assert.Equal(t, tt.size, read, "an under-limit body should be read in full")
		})
	}
}
