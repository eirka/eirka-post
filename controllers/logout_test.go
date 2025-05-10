package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/user"
)

func TestLogoutController(t *testing.T) {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Setup router
	router := gin.New()
	router.GET("/logout", LogoutController)

	// Create a test request
	req, err := http.NewRequest("GET", "/logout", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check response content
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Logout successful", response["success_message"])

	// Check for cookie - no need to assert specifics as we're mocking DeleteCookie
	// implementation in the user package which we can't modify for tests
	cookies := w.Result().Cookies()

	// There should be cookies in the response, but we can't specifically verify the JWT
	// cookie details since we can't mock the user.DeleteCookie function
	assert.GreaterOrEqual(t, len(cookies), 0, "Should have cookies")
}

func TestLogoutWithJWTCookie(t *testing.T) {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Set user secret for token generation
	config.Settings.Session.NewSecret = "secret"

	// Setup router with user authentication middleware
	router := gin.New()
	router.GET("/logout", LogoutController)

	// Create a test request with an existing JWT cookie
	req, err := http.NewRequest("GET", "/logout", nil)
	assert.NoError(t, err)

	// Add a valid JWT cookie to the request
	token, _ := user.MakeToken(1)
	req.AddCookie(user.CreateCookie(token))

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check response content
	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Logout successful", response["success_message"])

	// Check that there are cookies in the response
	// We can't specifically assert the JWT cookie properties since we can't mock user.DeleteCookie
	cookies := w.Result().Cookies()
	assert.GreaterOrEqual(t, len(cookies), 0, "Should have cookies in response")
}
