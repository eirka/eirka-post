package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStripNonAlpha(t *testing.T) {
	assert := assert.New(t)

	// Test with empty string
	assert.Equal("", stripNonAlpha(""), "should be equal")

	// Test with string with spaces
	assert.Equal("test", stripNonAlpha(" test "), "should be equal")

	// Test with string with weird characters
	assert.Equal("test", stripNonAlpha("t!e@s#t$"), "should be equal")

	// Test with string with weird characters and spaces
	assert.Equal("test", stripNonAlpha(" t!e@s#t$ "), "should be equal")

	// Test with a long string to ensure performance is good
	longString := strings.Repeat("test!@#$1234 ", 1000)
	result := stripNonAlpha(longString)
	assert.Equal(strings.Repeat("test1234", 1000), result, "should strip non-alphanumeric from long string")
}

// BenchmarkStripNonAlpha tests the performance of our stripNonAlpha function
func BenchmarkStripNonAlpha(b *testing.B) {
	// Create a test string with a mix of alphanumeric and non-alphanumeric chars
	longString := strings.Repeat("test!@#$1234 ", 1000)

	// Reset timer before the loop
	b.ResetTimer()

	// Run the stripNonAlpha function b.N times
	for i := 0; i < b.N; i++ {
		stripNonAlpha(longString)
	}
}

func TestContainsWords(t *testing.T) {
	assert := assert.New(t)

	// Test with empty string
	assert.False(containsWords("", wordPatterns...), "should be false")

	// Test with string with spaces
	assert.False(containsWords(" test ", wordPatterns...), "should be false")

	// Test with string with weird characters
	assert.False(containsWords("t!e@s#t$", wordPatterns...), "should be false")

	// Test with string with weird characters and spaces
	assert.False(containsWords(" t!e@s#t$ ", wordPatterns...), "should be false")

	// Test with string with weird characters and spaces
	assert.True(containsWords("loli", wordPatterns...), "should be true")

	// Test with string with spaces
	assert.True(containsWords(" loli ", wordPatterns...), "should be true")

	// Test with string with weird characters
	assert.True(containsWords(stripNonAlpha("l!o@l#i$"), wordPatterns...), "should be true")

	// Test with string with weird characters and spaces
	assert.True(containsWords(stripNonAlpha(" l!o@l#i$ "), wordPatterns...), "should be true")

	// Test words that might cause false positives (these will now match with our updated patterns)
	assert.False(containsWords("childhood", wordPatterns...), "should be false") // Contains 'child'
	assert.False(containsWords("skidding", wordPatterns...), "should be false")  // Contains 'kid'
	assert.False(containsWords("script", wordPatterns...), "should be false")    // Contains 'cp'
}

func TestSpamFilterEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with empty comment
	form := url.Values{}
	form.Add("comment", "")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.True(t, wasHandlerCalled, "Handler should be called for empty comments")
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestSpamFilterNormal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with normal comment
	form := url.Values{}
	form.Add("comment", "This is a normal comment")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.True(t, wasHandlerCalled, "Handler should be called for normal comments")
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestSpamFilterBannedWord(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with banned word
	form := url.Values{}
	form.Add("comment", "This comment contains loli which is banned")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.False(t, wasHandlerCalled, "Handler should not be called for comments with banned words")
	assert.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestSpamFilterBannedWordWithChars(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with banned word with characters
	form := url.Values{}
	form.Add("comment", "This comment contains l!o@l#i$ which is banned")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.False(t, wasHandlerCalled, "Handler should not be called for comments with banned words with characters")
	assert.NotEqual(t, http.StatusOK, recorder.Code)
}

func TestSpamFilterNormalUrl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with normal URL
	form := url.Values{}
	form.Add("comment", "Check out this link https://example.com/longer-page-path")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.True(t, wasHandlerCalled, "Handler should be called for comments with normal URLs")
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestSpamFilterUrlShortener(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup
	router := gin.New()
	recorder := httptest.NewRecorder()

	// Create a test request with URL shortener
	form := url.Values{}
	form.Add("comment", "Check out this link bit.ly/abc123")
	request, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Setup middleware
	var wasHandlerCalled bool
	router.Use(SpamFilter())
	router.POST("/", func(c *gin.Context) {
		wasHandlerCalled = true
		c.Status(http.StatusOK)
	})

	// Execute request
	router.ServeHTTP(recorder, request)

	// Validate
	assert.False(t, wasHandlerCalled, "Handler should not be called for comments with URL shorteners")
	assert.NotEqual(t, http.StatusOK, recorder.Code)
}
