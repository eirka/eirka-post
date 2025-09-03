package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func performRequestWithUserAgent(r http.Handler, method, path, userAgent string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func captureLogOutput(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	fn()
	log.SetOutput(os.Stderr) // Restore default output
	return buf.String()
}

func TestRequestLoggerSuccessfulPOST(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testUserAgent := "Mozilla/5.0 (compatible; TestBot/1.0)"

	logOutput := captureLogOutput(func() {
		w := performRequestWithUserAgent(router, "POST", "/test", testUserAgent)
		assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")
	})

	// Verify log output contains expected information
	assert.Contains(t, logOutput, "[POST SUCCESS]", "Log should contain POST SUCCESS marker")
	assert.Contains(t, logOutput, "POST /test", "Log should contain method and path")
	assert.Contains(t, logOutput, "Status: 200", "Log should contain status code")
	assert.Contains(t, logOutput, testUserAgent, "Log should contain user agent")
	assert.Contains(t, logOutput, "IP:", "Log should contain IP information")
}

func TestRequestLoggerSuccessfulPOSTWithRedirect(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.POST("/redirect-test", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/success")
	})

	testUserAgent := "curl/7.68.0"

	logOutput := captureLogOutput(func() {
		w := performRequestWithUserAgent(router, "POST", "/redirect-test", testUserAgent)
		assert.Equal(t, http.StatusSeeOther, w.Code, "Should return 303 See Other")
	})

	// Verify successful redirect is logged (3xx is success)
	assert.Contains(t, logOutput, "[POST SUCCESS]", "Log should contain POST SUCCESS marker for redirect")
	assert.Contains(t, logOutput, "POST /redirect-test", "Log should contain method and path")
	assert.Contains(t, logOutput, "Status: 303", "Log should contain redirect status code")
	assert.Contains(t, logOutput, testUserAgent, "Log should contain user agent")
}

func TestRequestLoggerFailedPOST(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.POST("/fail", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	testUserAgent := "BadBot/1.0"

	logOutput := captureLogOutput(func() {
		w := performRequestWithUserAgent(router, "POST", "/fail", testUserAgent)
		assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 Bad Request")
	})

	// Verify failed requests are NOT logged
	assert.NotContains(t, logOutput, "[POST SUCCESS]", "Log should not contain POST SUCCESS marker for failed requests")
	assert.NotContains(t, logOutput, testUserAgent, "Log should not contain user agent for failed requests")
}

func TestRequestLoggerGETRequest(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testUserAgent := "Mozilla/5.0 (compatible; GetBot/1.0)"

	logOutput := captureLogOutput(func() {
		w := performRequestWithUserAgent(router, "GET", "/test", testUserAgent)
		assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")
	})

	// Verify GET requests are NOT logged (only POST requests should be logged)
	assert.NotContains(t, logOutput, "[POST SUCCESS]", "Log should not contain POST SUCCESS marker for GET requests")
	assert.NotContains(t, logOutput, testUserAgent, "Log should not contain user agent for GET requests")
}

func TestRequestLoggerEmptyUserAgent(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	logOutput := captureLogOutput(func() {
		w := performRequestWithUserAgent(router, "POST", "/test", "") // Empty user agent
		assert.Equal(t, http.StatusOK, w.Code, "Should return 200 OK")
	})

	// Verify empty user agent defaults to "Unknown"
	assert.Contains(t, logOutput, "[POST SUCCESS]", "Log should contain POST SUCCESS marker")
	assert.Contains(t, logOutput, "User-Agent: Unknown", "Log should show Unknown for empty user agent")
}

func TestRequestLoggerMultipleRequests(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(RequestLogger())

	router.POST("/test1", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	router.POST("/test2", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	router.GET("/test3", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "get success"})
	})

	userAgent1 := "Bot1/1.0"
	userAgent2 := "Bot2/1.0"
	userAgent3 := "Bot3/1.0"

	logOutput := captureLogOutput(func() {
		// Successful POST - should be logged
		w1 := performRequestWithUserAgent(router, "POST", "/test1", userAgent1)
		assert.Equal(t, http.StatusCreated, w1.Code)

		// Failed POST - should NOT be logged
		w2 := performRequestWithUserAgent(router, "POST", "/test2", userAgent2)
		assert.Equal(t, http.StatusInternalServerError, w2.Code)

		// Successful GET - should NOT be logged
		w3 := performRequestWithUserAgent(router, "GET", "/test3", userAgent3)
		assert.Equal(t, http.StatusOK, w3.Code)
	})

	// Verify only the successful POST is logged
	assert.Contains(t, logOutput, userAgent1, "Should log user agent for successful POST")
	assert.Contains(t, logOutput, "Status: 201", "Should log 201 status for successful POST")
	assert.NotContains(t, logOutput, userAgent2, "Should not log user agent for failed POST")
	assert.NotContains(t, logOutput, userAgent3, "Should not log user agent for successful GET")

	// Count occurrences of POST SUCCESS
	successCount := strings.Count(logOutput, "[POST SUCCESS]")
	assert.Equal(t, 1, successCount, "Should only log one successful POST")
}
