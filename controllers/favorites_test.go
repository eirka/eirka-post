package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
)

func TestFavoritesController(t *testing.T) {
	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a middleware that sets user data for our test
	router.Use(func(c *gin.Context) {
		// Set user data in the context
		c.Set("userdata", user.User{
			ID: 2,
		})
		c.Next()
	})

	router.POST("/favorite", FavoritesController)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Set up mock for adding a favorite when not already favorited
	mock.ExpectBegin()
	// Image exists
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)

	// No favorite exists
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites WHERE image_id = \? AND user_id = \? FOR UPDATE`).
		WithArgs(1, 2).
		WillReturnRows(rows)
	mock.ExpectRollback()

	// Post begins a new transaction
	mock.ExpectBegin()
	// Image exists
	rows = sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)

	// No favorite exists
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites WHERE image_id = \? AND user_id = \?`).
		WithArgs(1, 2).
		WillReturnRows(rows)

	mock.ExpectExec("INSERT into favorites").
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// test adding a favorite
	request := []byte(`{"image": 1}`)

	req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "HTTP request code should match")
	assert.Contains(t, w.Body.String(), audit.AuditFavoriteAdded, "Response should contain success message")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestFavoritesControllerRemove(t *testing.T) {
	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a middleware that sets user data for our test
	router.Use(func(c *gin.Context) {
		// Set user data in the context
		c.Set("userdata", user.User{
			ID: 2,
		})
		c.Next()
	})

	router.POST("/favorite", FavoritesController)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Set up mock for removing a favorite that is already favorited
	mock.ExpectBegin()
	// Image exists
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)

	// Favorite exists
	rows = sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites WHERE image_id = \? AND user_id = \? FOR UPDATE`).
		WithArgs(1, 2).
		WillReturnRows(rows)

	// Delete the favorite
	mock.ExpectExec("DELETE FROM favorites").
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// test removing a favorite
	request := []byte(`{"image": 1}`)

	req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "HTTP request code should match")
	assert.Contains(t, w.Body.String(), audit.AuditFavoriteRemoved, "Response should contain success message")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestFavoritesControllerBadInput(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a middleware that sets user data for our test
	router.Use(func(c *gin.Context) {
		// Set user data in the context
		c.Set("userdata", user.User{
			ID: 2,
		})
		c.Next()
	})

	router.POST("/favorite", FavoritesController)

	badRequests := [][]byte{
		[]byte(`{}`),                  // empty
		[]byte(`{"image": "string"}`), // wrong type
		[]byte(`{"image": 0}`),        // zero value
		[]byte(`{"image": -1}`),       // zero value
	}

	for _, request := range badRequests {
		req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(request))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "127.0.0.1")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code, "HTTP request code should match")
	}

	badRequest := []byte(`{"image": 1, "extra": "stuff"}`) // extra field

	req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(badRequest))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code, "HTTP request code should match")

}

func TestFavoritesControllerImageNotFound(t *testing.T) {
	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a middleware that sets user data for our test
	router.Use(func(c *gin.Context) {
		// Set user data in the context
		c.Set("userdata", user.User{
			ID: 2,
		})
		c.Next()
	})

	router.POST("/favorite", FavoritesController)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Set up mock for a non-existent image
	mock.ExpectBegin()
	// Image does not exist
	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)
	mock.ExpectRollback()

	// test with non-existent image
	request := []byte(`{"image": 1}`)

	req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code, "HTTP request code should match")
	assert.Contains(t, w.Body.String(), e.ErrNotFound.Error(), "Response should contain error message")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestFavoritesControllerPostImageNotFound(t *testing.T) {
	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"

	// Use a middleware that sets user data for our test
	router.Use(func(c *gin.Context) {
		// Set user data in the context
		c.Set("userdata", user.User{
			ID: 2,
		})
		c.Next()
	})

	router.POST("/favorite", FavoritesController)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Set up mock for images existing in Status but not in Post
	mock.ExpectBegin()
	// Image exists for initial check
	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)

	// No favorite exists
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM favorites WHERE image_id = \? AND user_id = \? FOR UPDATE`).
		WithArgs(1, 2).
		WillReturnRows(rows)
	mock.ExpectRollback()

	// Post begins a new transaction
	mock.ExpectBegin()
	// Image suddenly doesn't exist
	rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM images WHERE image_id = \?`).
		WithArgs(1).
		WillReturnRows(rows)
	mock.ExpectRollback()

	// test with non-existent image
	request := []byte(`{"image": 1}`)

	req, _ := http.NewRequest("POST", "/favorite", bytes.NewBuffer(request))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", "127.0.0.1")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code, "HTTP request code should match")
	assert.Contains(t, w.Body.String(), e.ErrNotFound.Error(), "Response should contain error message")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}
