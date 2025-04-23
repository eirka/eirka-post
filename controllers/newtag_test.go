package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"
)

func TestNewTagController(t *testing.T) {
	var err error

	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)
	user.Secret = "secret"

	// Create router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	router.POST("/newtag", NewTagController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	// Set up mock database
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Check if tag already exists (it doesn't)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM tags WHERE ib_id = \? AND tag_name = \? FOR UPDATE`).
		WithArgs(1, "example tag").
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Tag insert transaction
	mock.ExpectBegin()
	mock.ExpectExec("INSERT into tags").
		WithArgs("example tag", 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Audit log
	mock.ExpectExec("INSERT INTO audit").
		WithArgs(2, 1, audit.BoardLog, "127.0.0.1", audit.AuditNewTag, "example tag").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Setup Redis mock for key deletion
	redis.Cache.Mock.Command("DEL", "tags:1")

	// Create new tag request with valid data
	reqBody := map[string]interface{}{
		"ib":   1,
		"name": "example tag",
		"type": 1,
	}
	
	jsonBytes, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/newtag", bytes.NewBuffer(jsonBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, successMessage(audit.AuditNewTag), w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewTagControllerDuplicate(t *testing.T) {
	var err error

	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)
	user.Secret = "secret"

	// Create router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	router.POST("/newtag", NewTagController)

	// Set up mock database
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Check if tag already exists (it does)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM tags WHERE ib_id = \? AND tag_name = \? FOR UPDATE`).
		WithArgs(1, "duplicate tag").
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Create new tag request with duplicate tag name
	reqBody := map[string]interface{}{
		"ib":   1,
		"name": "duplicate tag",
		"type": 1,
	}
	
	jsonBytes, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/newtag", bytes.NewBuffer(jsonBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, errorMessage(e.ErrDuplicateTag), w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewTagControllerRedisError(t *testing.T) {
	var err error

	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)
	user.Secret = "secret"

	// Create router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	router.POST("/newtag", NewTagController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	// Set up mock database
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Check if tag already exists (it doesn't)
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM tags WHERE ib_id = \? AND tag_name = \? FOR UPDATE`).
		WithArgs(1, "example tag").
		WillReturnRows(rows)
	mock.ExpectCommit()

	// Tag insert transaction
	mock.ExpectBegin()
	mock.ExpectExec("INSERT into tags").
		WithArgs("example tag", 1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Setup Redis mock for key deletion with error
	redisKey := fmt.Sprintf("%s:%d", "tags", 1)
	redis.Cache.Mock.Command("DEL", redisKey).ExpectError(errors.New("redis error"))

	// Create new tag request with valid data
	reqBody := map[string]interface{}{
		"ib":   1,
		"name": "example tag",
		"type": 1,
	}
	
	jsonBytes, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/newtag", bytes.NewBuffer(jsonBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response - should fail with internal error due to Redis error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, errorMessage(e.ErrInternalError), w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewTagControllerInvalidParams(t *testing.T) {
	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)
	user.Secret = "secret"

	// Create router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	router.POST("/newtag", NewTagController)

	// Test cases for invalid parameters
	testCases := []struct {
		name  string
		body  map[string]interface{}
		code  int
	}{
		{
			"missing_name", 
			map[string]interface{}{
				"ib": 1,
				"type": 1,
			},
			http.StatusBadRequest,
		},
		{
			"missing_type", 
			map[string]interface{}{
				"ib": 1,
				"name": "example tag",
			},
			http.StatusBadRequest,
		},
		{
			"missing_ib", 
			map[string]interface{}{
				"name": "example tag",
				"type": 1,
			},
			http.StatusBadRequest,
		},
		{
			"tag_too_short", 
			map[string]interface{}{
				"ib": 1,
				"name": "a",  // Assuming min length is more than 1
				"type": 1,
			},
			http.StatusBadRequest,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create JSON request body
			jsonBytes, err := json.Marshal(tc.body)
			assert.NoError(t, err)
			
			// Create HTTP request
			req, err := http.NewRequest("POST", "/newtag", bytes.NewBuffer(jsonBytes))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", "127.0.0.1")
			
			// Create response recorder
			w := httptest.NewRecorder()
			
			// Perform the request
			router.ServeHTTP(w, req)
			
			// Check response code
			assert.Equal(t, tc.code, w.Code)
			// Verify error response contains error_message
			assert.Contains(t, w.Body.String(), "error_message")
		})
	}
}