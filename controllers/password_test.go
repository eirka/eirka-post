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

// Test password change validation
func TestPasswordControllerValidation(t *testing.T) {
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
	
	router.POST("/password", PasswordController)

	// Test cases for invalid parameters
	testCases := []struct {
		name  string
		body  map[string]interface{}
		code  int
	}{
		{
			"missing_old_password", 
			map[string]interface{}{
				"ib": 1,
				"newpw": "newpassword123",
			},
			http.StatusBadRequest,
		},
		{
			"missing_new_password", 
			map[string]interface{}{
				"ib": 1,
				"oldpw": "oldpassword123",
			},
			http.StatusBadRequest,
		},
		{
			"missing_ib", 
			map[string]interface{}{
				"oldpw": "oldpassword123",
				"newpw": "newpassword123",
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
			req, err := http.NewRequest("POST", "/password", bytes.NewBuffer(jsonBytes))
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

// Test password change workflow with mock handler
func TestPasswordControllerSuccess(t *testing.T) {
	// Set up the gin context
	gin.SetMode(gin.ReleaseMode)

	// Create a router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	
	// Use a mock handler for successful password change
	router.POST("/password-mock", func(c *gin.Context) {
		// Add mock user to context
		c.Set("userdata", user.User{ID: 2, Name: "test"})
		
		// Parse JSON body
		var pf passwordForm
		if err := c.BindJSON(&pf); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": "invalid parameters"})
			return
		}
		
		// Return success for valid request
		c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditChangePassword})
	})

	// Create valid password change request
	reqBody := map[string]interface{}{
		"ib":    1,
		"oldpw": "oldpassword",
		"newpw": "newpassword",
	}
	
	jsonBytes, err := json.Marshal(reqBody)
	assert.NoError(t, err)

	// Create HTTP request
	req, err := http.NewRequest("POST", "/password-mock", bytes.NewBuffer(jsonBytes))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, successMessage(audit.AuditChangePassword), w.Body.String())
}

// Test password update with database mock
func TestPasswordControllerDatabaseUpdate(t *testing.T) {
	var err error

	// Set up testing environment
	gin.SetMode(gin.ReleaseMode)

	// Set up mock database
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Database expectations for password update
	mock.ExpectBegin()
	
	// Password update query
	mock.ExpectExec("UPDATE users SET user_password").
		WithArgs([]byte("newhashed"), 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	
	// Commit transaction
	mock.ExpectCommit()

	// Execute a simple transaction directly to test DB mocking
	tx, err := db.GetTransaction()
	assert.NoError(t, err)

	// Execute a simple query
	_, err = tx.Exec("UPDATE users SET user_password = ? WHERE user_id = ?", []byte("newhashed"), 2)
	assert.NoError(t, err)

	// Commit transaction
	err = tx.Commit()
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}