package controllers

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"
)

// gin router for tests
var router *gin.Engine

func init() {
	user.Secret = "secret"

	// Set up fake Redis connection
	redis.NewRedisMock()

	gin.SetMode(gin.ReleaseMode)

	router = gin.Default()

	router.Use(user.Auth(false))

	router.POST("/tag/add", AddTagController)
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performJsonRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorMessage(err error) string {
	return fmt.Sprintf(`{"error_message":"%s"}`, err)
}

func successMessage(message string) string {
	return fmt.Sprintf(`{"success_message":"%s"}`, message)
}

func TestAddTagController(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	duperows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`select count\(1\) from tagmap`).WillReturnRows(duperows)

	mock.ExpectExec("INSERT into tagmap").
		WithArgs(1, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO audit \(user_id,ib_id,audit_type,audit_ip,audit_time,audit_action,audit_info\)`).
		WithArgs(1, 1, audit.BoardLog, "127.0.0.1", audit.AuditAddTag, "1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	redis.RedisCache.Mock.Command("DEL", "tags:1", "tag:1:1", "image:1")

	request := []byte(`{"ib": 1, "tag": 1, "image": 1}`)

	first := performJsonRequest(router, "POST", "/tag/add", request)

	assert.Equal(t, first.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), successMessage(audit.AuditAddTag), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagControllerBadInput(t *testing.T) {

	var reuesttests = []struct {
		name string
		in   []byte
	}{
		{"nofield", []byte(`{}`)},
		{"badfield", []byte(`{"derp": 1}`)},
		{"badmissing", []byte(`{"ib": 0}`)},
		{"badmissing", []byte(`{"ib": 0, "tag": 1}`)},
		{"badmissing", []byte(`{"image": 1}`)},
		{"badib", []byte(`{"ib": 0, "tag": 1, "image": 1}`)},
		{"badib", []byte(`{"ib": dur, "tag": 1, "image": 1}`)},
		{"badtag", []byte(`{"ib": 1, "tag": 0, "image": 1}`)},
		{"badtag", []byte(`{"ib": 1, "tag": dur, "image": 1}`)},
		{"badimage", []byte(`{"ib": 1, "tag": 1, "image": 0}`)},
		{"badimage", []byte(`{"ib": 1, "tag": 1, "image": dur}`)},
		{"badall", []byte(`{"ib": 0, "tag": 0, "image": 0}`)},
	}

	for _, test := range reuesttests {
		first := performJsonRequest(router, "POST", "/tag/add", test.in)
		assert.Equal(t, first.Code, 400, fmt.Sprintf("HTTP request code should match for request %s", test.name))
	}

}

func TestAddTagControllerImageNotFound(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	request := []byte(`{"ib": 1, "tag": 1, "image": 1}`)

	first := performJsonRequest(router, "POST", "/tag/add", request)

	assert.Equal(t, first.Code, 400, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), errorMessage(e.ErrNotFound), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestAddTagControllerDuplicate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	statusrows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(1\) FROM images`).WillReturnRows(statusrows)

	duperows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`select count\(1\) from tagmap`).WillReturnRows(duperows)

	request := []byte(`{"ib": 1, "tag": 1, "image": 1}`)

	first := performJsonRequest(router, "POST", "/tag/add", request)

	assert.Equal(t, first.Code, 400, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), errorMessage(e.ErrDuplicateTag), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
