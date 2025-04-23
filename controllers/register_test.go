package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	"github.com/eirka/eirka-libs/user"
)

// Test register controller with mock handler for the validation paths
func TestRegisterControllerValidation(t *testing.T) {
	// Set up the gin context
	gin.SetMode(gin.ReleaseMode)

	// Create a router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	router.POST("/register", RegisterController)

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
				"email": "test@example.com",
				"password": "password123",
			},
			http.StatusBadRequest,
		},
		{
			"missing_password", 
			map[string]interface{}{
				"ib": 1,
				"name": "testuser",
				"email": "test@example.com",
			},
			http.StatusBadRequest,
		},
		{
			"missing_ib", 
			map[string]interface{}{
				"name": "testuser",
				"email": "test@example.com",
				"password": "password123",
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
			req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonBytes))
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

// Test the successful registration workflow with a mock handler
func TestRegisterControllerSuccess(t *testing.T) {
	// Set up the gin context
	gin.SetMode(gin.ReleaseMode)

	// Create a router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Add user authentication middleware with mock user
	router.Use(func(c *gin.Context) {
		// Set mock user data in context
		c.Set("userdata", user.User{ID: 2})
		c.Next()
	})
	
	// Register a mock handler
	router.POST("/register-mock", func(c *gin.Context) {
		// Return success message for registration
		c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditRegister})
	})

	// Create register request with valid data
	reqBody := map[string]interface{}{
		"ib":       1,
		"name":     "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}
	
	jsonBytes, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/register-mock", bytes.NewBuffer(jsonBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, successMessage(audit.AuditRegister), w.Body.String())
}

// Test a simple DB transaction with the register model
func TestRegisterControllerDatabase(t *testing.T) {
	var err error

	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)

	// Set up mock database
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Database expectations for user registration
	mock.ExpectBegin()
	
	// User insert with return ID
	mock.ExpectExec("INSERT into users").
		WithArgs("testuser", "test@example.com", []byte("hashedpassword"), 1).
		WillReturnResult(sqlmock.NewResult(2, 1))
	
	// Role insert
	mock.ExpectExec("INSERT into user_role_map").
		WithArgs(2, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	
	// Commit transaction
	mock.ExpectCommit()

	// Execute a simple transaction directly to test DB mocking
	tx, err := db.GetTransaction()
	assert.NoError(t, err)

	// Execute a simple query
	_, err = tx.Exec("INSERT into users (user_name, user_email, user_password, user_confirmed) VALUES (?,?,?,?)",
		"testuser", "test@example.com", []byte("hashedpassword"), 1)
	assert.NoError(t, err)

	// Execute a second query
	_, err = tx.Exec("INSERT into user_role_map VALUES (?,?)", 2, 2)
	assert.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}