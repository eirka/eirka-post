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

	router = gin.New()

	users.Use(user.Auth(true))

	router.POST("/email", EmailController)
}

func performJwtJsonRequest(r http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("X-Real-Ip", "123.0.0.1")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performJwtFormRequest(r http.Handler, method, path, token string, body bytes.Buffer) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, &body)
	req.Header.Set("X-Real-Ip", "123.0.0.1")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEmailController(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	rows := sqlmock.NewRows([]string{"name", "email"}).AddRow("test", "old@test.com")
	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnRows(rows)

	mock.ExpectExec("UPDATE users SET user_email").
		WithArgs("cool@test.com", 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := []byte(`{"ib": 1, "email": "test@test.com"}`)

	first := performJsonRequest(router, "POST", "/email", request)

	assert.Equal(t, first.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
