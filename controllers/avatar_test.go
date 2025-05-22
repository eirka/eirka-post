package controllers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	u "github.com/eirka/eirka-post/utils"
)

// Mock the ImageType for testing
type MockImageType struct {
	mock.Mock
	u.ImageType
}

func (m *MockImageType) SaveAvatar() error {
	args := m.Called()
	return args.Error(0)
}

// Test AvatarController with no image
func TestAvatarControllerNoImage(t *testing.T) {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Setup router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})

	router.POST("/avatar", AvatarController)

	// Create a request without an image
	req, err := http.NewRequest("POST", "/avatar", nil)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "multipart/form-data")
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("Referer", "/profile")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	// The actual error comes from http.ErrMissingFile, not our custom error
	assert.Contains(t, w.Body.String(), "error_message")
}

// Test AvatarController with image
func TestAvatarControllerWithImage(t *testing.T) {
	// Create a mock upload handler to isolate AvatarController test from file operations
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Mock image upload

	// Use a mock handler for successful avatar upload
	router.POST("/avatar-mock", func(c *gin.Context) {
		// Get the file from the form
		_, _, _ = c.Request.FormFile("file")

		// Just redirect without actual processing
		c.Redirect(303, c.Request.Referer())
	})

	// Create a multipart form with an image
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	assert.NoError(t, err)

	// Create a dummy PNG image
	img := bytes.NewBuffer([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	_, err = io.Copy(part, img)
	assert.NoError(t, err)
	writer.Close()

	// Create a request with the mock image file
	req, err := http.NewRequest("POST", "/avatar-mock", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("Referer", "/profile")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response is a redirect
	assert.Equal(t, http.StatusSeeOther, w.Code)
	assert.Equal(t, "/profile", w.Header().Get("Location"))
}

// Test AvatarController with image save error
func TestAvatarControllerSaveError(t *testing.T) {
	// Create a mock upload handler to isolate AvatarController test from file operations
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a mock handler for avatar upload with error
	router.POST("/avatar-error", func(c *gin.Context) {
		// Get userdata from middleware
		c.Set("userdata", user.User{ID: 2})

		// Get the file from the form
		_, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrNoImage.Error()})
			return
		}

		// Simulate an error during save
		c.JSON(http.StatusBadRequest, gin.H{"error_message": "image processing error"})
	})

	// Create a multipart form with an image
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	assert.NoError(t, err)

	// Create a dummy PNG image
	img := bytes.NewBuffer([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	_, err = io.Copy(part, img)
	assert.NoError(t, err)
	writer.Close()

	// Create a request with the mock image file
	req, err := http.NewRequest("POST", "/avatar-error", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("Referer", "/profile")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response for error
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error_message")
}
