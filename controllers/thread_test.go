package controllers

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
)

// TestThreadControllerNoImage tests validation error when no image is provided
func TestThreadControllerNoImage(t *testing.T) {
	config.Settings.Session.NewSecret = "secret"

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/thread", ThreadController)

	// Prepare test parameters with no file
	params := map[string]string{
		"ib":      "1",
		"title":   "test thread",
		"comment": "test comment",
	}

	// Perform the request without a file
	resp := performRequestWithFileAndParams(router, "POST", "/thread", "file", "", nil, params)

	// Check results
	assert.Equal(t, 400, resp.Code, "HTTP error code should match")
	assert.JSONEq(t, errorMessage(e.ErrNoImage), resp.Body.String(), "Error message should match")
}

// TestThreadControllerInvalidParams tests validation errors for missing or invalid parameters
func TestThreadControllerInvalidParams(t *testing.T) {
	config.Settings.Session.NewSecret = "secret"

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/thread", ThreadController)

	// Create a test file - content doesn't matter since we'll fail earlier
	fileContents := []byte{0xff, 0xd8, 0xff, 0xe0}

	// Test cases for missing or invalid parameters
	testCases := []struct {
		name   string
		params map[string]string
	}{
		{"missing_title", map[string]string{"ib": "1", "comment": "test comment"}},
		{"missing_comment", map[string]string{"ib": "1", "title": "test thread"}},
		{"missing_ib", map[string]string{"title": "test thread", "comment": "test comment"}},
		{"invalid_ib", map[string]string{"ib": "abc", "title": "test thread", "comment": "test comment"}},
		{"ib_zero", map[string]string{"ib": "0", "title": "test thread", "comment": "test comment"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequestWithFileAndParams(router, "POST", "/thread", "file", "test.jpg", fileContents, tc.params)
			assert.Equal(t, 400, resp.Code, "HTTP error code should match")
			// The specific error message depends on the validation logic
			assert.True(t, strings.Contains(resp.Body.String(), "error_message"),
				"Response should contain an error message")
		})
	}
}

// TestThreadControllerValidationErrors tests validation errors for invalid title or comment
func TestThreadControllerValidationErrors(t *testing.T) {
	config.Settings.Session.NewSecret = "secret"

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/thread", ThreadController)

	// Create a test file
	fileContents := []byte{0xff, 0xd8, 0xff, 0xe0}

	// Test cases for title/comment validation errors
	testCases := []struct {
		name          string
		params        map[string]string
		expectedError error
	}{
		{
			"title_too_short",
			map[string]string{"ib": "1", "title": "a", "comment": "valid comment"},
			e.ErrTitleShort,
		},
		{
			"title_too_long",
			map[string]string{"ib": "1", "title": strings.Repeat("a", 1000), "comment": "valid comment"},
			e.ErrTitleLong,
		},
		{
			"comment_too_short",
			map[string]string{"ib": "1", "title": "valid title", "comment": "a"},
			e.ErrCommentShort,
		},
		{
			"comment_too_long",
			map[string]string{"ib": "1", "title": "valid title", "comment": strings.Repeat("a", 10000)},
			e.ErrCommentLong,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequestWithFileAndParams(router, "POST", "/thread", "file", "test.jpg", fileContents, tc.params)
			assert.Equal(t, 400, resp.Code, "HTTP error code should match")
			assert.JSONEq(t, errorMessage(tc.expectedError), resp.Body.String(), "Error message should match")
		})
	}
}
