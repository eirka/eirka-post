package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUptimeController(t *testing.T) {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Setup router
	router := gin.New()
	router.GET("/uptime", UptimeController)

	// Store the original startTime and restore it after test
	originalStartTime := startTime
	defer func() { startTime = originalStartTime }()

	// Set start time to a known value (10 minutes ago)
	startTime = time.Now().Add(-10 * time.Minute)

	// Create a test request
	req, err := http.NewRequest("GET", "/uptime", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check response has uptime string
	assert.Contains(t, w.Body.String(), "uptime")

	// Check response has expected uptime value (should be around 10m)
	assert.Contains(t, w.Body.String(), "10m")

	// Set start time to a known value (65 minutes ago)
	startTime = time.Now().Add(-65 * time.Minute)

	// Create a new response recorder
	w = httptest.NewRecorder()

	// Perform the request again
	router.ServeHTTP(w, req)

	// Check response still has correct status
	assert.Equal(t, http.StatusOK, w.Code)

	// Check response has expected uptime value (should be around 65m)
	assert.Contains(t, w.Body.String(), "65m")
}
