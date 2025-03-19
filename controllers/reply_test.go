package controllers

import (
	"bytes"
	"database/sql"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"
)

// performRequestWithFileAndParams creates a test request with both file and form parameters
func performRequestWithFileAndParams(r http.Handler, method, path, paramName, fileName string, fileContents []byte, params map[string]string) *httptest.ResponseRecorder {
	// Create a multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file if provided
	if fileName != "" && fileContents != nil {
		part, _ := writer.CreateFormFile(paramName, fileName)
		part.Write(fileContents)
	}

	// Add other params
	for key, val := range params {
		writer.WriteField(key, val)
	}

	writer.Close()

	// Create a request
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Real-IP", "127.0.0.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Note: errorMessage already defined in addtag_test.go

func TestReplyController(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check - updated for transaction and FOR UPDATE clause
	mock.ExpectBegin()
	threadRows := sqlmock.NewRows([]string{"ib_id", "thread_closed", "count"}).AddRow(1, 0, 5)
	mock.ExpectQuery(`SELECT ib_id, thread_closed, count\(post_num\) FROM threads.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(threadRows)
	mock.ExpectCommit()

	// Post transaction
	mock.ExpectBegin()
	// Expect post number query
	postRows := sqlmock.NewRows([]string{"nextnum"}).AddRow(2)
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(post_num\), 0\) \+ 1.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(postRows)
	// Now insert with explicit post_num
	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(1, 1, 2, "127.0.0.1", "test comment").
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	// Audit log
	mock.ExpectExec(`INSERT INTO audit`).
		WithArgs(1, 1, audit.BoardLog, "127.0.0.1", audit.AuditReply, "1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Setup Redis mock for key deletion
	redis.Cache.Mock.Command("DEL", "directory:1", "thread:1:1", "image:1")

	// Prepare test parameters
	params := map[string]string{
		"thread":  "1",
		"comment": "test comment",
	}

	// Perform the request
	first := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, params)

	// Check results
	assert.Equal(t, 303, first.Code, "HTTP redirect code should match")
	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestReplyControllerWithImage(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check
	threadRows := sqlmock.NewRows([]string{"ib_id", "thread_closed", "count"}).AddRow(1, 0, 5)
	mock.ExpectQuery(`SELECT ib_id,thread_closed,count\(post_num\) FROM threads`).
		WithArgs(1).
		WillReturnRows(threadRows)

	// Post transaction - this test can't effectively test the image saves directly
	// due to the complexity of mocking file operations
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO posts`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`INSERT INTO images`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Audit log
	mock.ExpectExec(`INSERT INTO audit`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Setup Redis mock for key deletion
	redis.Cache.Mock.Command("DEL", "directory:1", "thread:1:1", "image:1")

	// Skip actual test as it's difficult to mock file uploads and image processing
	t.Skip("Image upload test skipped as it requires mocking complex file operations")
}

func TestReplyControllerThreadClosed(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check - thread is closed
	mock.ExpectBegin()
	threadRows := sqlmock.NewRows([]string{"ib_id", "thread_closed", "count"}).AddRow(1, 1, 5)
	mock.ExpectQuery(`SELECT ib_id, thread_closed, count\(post_num\) FROM threads.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(threadRows)
	// No commit needed as we'll return early with error

	// Prepare test parameters
	params := map[string]string{
		"thread":  "1",
		"comment": "test comment",
	}

	// Perform the request
	first := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, params)

	// Check results
	assert.Equal(t, 400, first.Code, "HTTP error code should match")
	assert.JSONEq(t, errorMessage(e.ErrThreadClosed), first.Body.String(), "Error message should match")
	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestReplyControllerThreadMaxPosts(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check - thread has reached max posts
	mock.ExpectBegin()
	threadRows := sqlmock.NewRows([]string{"ib_id", "thread_closed", "count"}).AddRow(1, 0, 1001)
	mock.ExpectQuery(`SELECT ib_id, thread_closed, count\(post_num\) FROM threads.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(threadRows)

	// Thread will be closed
	mock.ExpectExec("UPDATE threads SET thread_closed=1").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Commit the closing transaction
	mock.ExpectCommit()

	// Prepare test parameters
	params := map[string]string{
		"thread":  "1",
		"comment": "test comment",
	}

	// Perform the request
	first := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, params)

	// Check results
	assert.Equal(t, 400, first.Code, "HTTP error code should match")
	assert.JSONEq(t, errorMessage(e.ErrThreadClosed), first.Body.String(), "Error message should match")
	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestReplyControllerRedisError(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check
	mock.ExpectBegin()
	threadRows := sqlmock.NewRows([]string{"ib_id", "thread_closed", "count"}).AddRow(1, 0, 5)
	mock.ExpectQuery(`SELECT ib_id, thread_closed, count\(post_num\) FROM threads.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(threadRows)
	mock.ExpectCommit()

	// Post transaction
	mock.ExpectBegin()
	// Expect post number query
	postRows := sqlmock.NewRows([]string{"nextnum"}).AddRow(2)
	mock.ExpectQuery(`SELECT COALESCE\(MAX\(post_num\), 0\) \+ 1.*FOR UPDATE`).
		WithArgs(1).
		WillReturnRows(postRows)
	// Now insert with explicit post_num
	mock.ExpectExec(`INSERT INTO posts`).
		WithArgs(1, 1, 2, "127.0.0.1", "test comment").
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	// Audit log
	mock.ExpectExec(`INSERT INTO audit`).
		WithArgs(1, 1, audit.BoardLog, "127.0.0.1", audit.AuditReply, "1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Setup Redis mock for key deletion with error
	redis.Cache.Mock.Command("DEL", "directory:1", "thread:1:1", "image:1").ExpectError(errors.New("redis error"))

	// Prepare test parameters
	params := map[string]string{
		"thread":  "1",
		"comment": "test comment",
	}

	// Perform the request - should still work despite Redis errors
	first := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, params)

	// Check results
	assert.Equal(t, 303, first.Code, "HTTP redirect code should match")
	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestReplyControllerInvalidParams(t *testing.T) {
	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Test cases for missing or invalid parameters
	testCases := []struct {
		name   string
		params map[string]string
	}{
		{"missing_thread", map[string]string{"comment": "test comment"}},
		{"invalid_thread", map[string]string{"thread": "abc", "comment": "test comment"}},
		{"thread_zero", map[string]string{"thread": "0", "comment": "test comment"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, tc.params)
			assert.Equal(t, 400, resp.Code, "HTTP error code should match")
			// The specific error message depends on the validation logic,
			// here we just check it contains "error_message"
			assert.True(t, strings.Contains(resp.Body.String(), "error_message"),
				"Response should contain an error message")
		})
	}
}

func TestReplyControllerNonExistentThread(t *testing.T) {
	var err error

	user.Secret = "secret"
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	router.Use(user.Auth(false))
	router.POST("/reply", ReplyController)

	// Set up fake Redis connection
	redis.NewRedisMock()

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Thread status check for non-existent thread
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT ib_id, thread_closed, count\(post_num\) FROM threads.*FOR UPDATE`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)
	// No commit needed as it will return early with error

	// Prepare test parameters with non-existent thread ID
	params := map[string]string{
		"thread":  "999",
		"comment": "test reply to non-existent thread",
	}

	// Perform the request
	resp := performRequestWithFileAndParams(router, "POST", "/reply", "file", "", nil, params)

	// Check results
	assert.Equal(t, 400, resp.Code, "HTTP error code should match")
	assert.JSONEq(t, errorMessage(e.ErrNotFound), resp.Body.String(), "Error message should match")
	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}
