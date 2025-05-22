package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	e "github.com/eirka/eirka-libs/errors"
)

func TestErrorController(t *testing.T) {
	// Set gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Setup router
	router := gin.New()
	router.GET("/error", ErrorController)

	// Create a test request
	req, err := http.NewRequest("GET", "/error", nil)
	assert.NoError(t, err)

	// Create a response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response code
	code, _ := e.ErrorMessage(e.ErrNotFound)
	assert.Equal(t, code, w.Code, "HTTP status code should match error code")

	// Check response content
	assert.JSONEq(t, errorMessage(e.ErrNotFound), w.Body.String(), "Error message should match")
}
