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

	//e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
)

func performJWTRequest(r http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-Ip", "123.0.0.1")
	req.AddCookie(user.CreateCookie(token))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEmailController(t *testing.T) {

	var err error

	user.Secret = "secret"

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(user.Auth(true))

	router.POST("/email", EmailController)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	rows := sqlmock.NewRows([]string{"name", "email"}).AddRow("test", "old@test.com")
	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnRows(rows)

	mock.ExpectBegin()
	userRows := sqlmock.NewRows([]string{"1"}).AddRow(1)
	mock.ExpectQuery(`SELECT 1 FROM users WHERE user_id = \? FOR UPDATE`).
		WithArgs(2).
		WillReturnRows(userRows)
	mock.ExpectExec("UPDATE users SET user_email").
		WithArgs("cool@test.com", 2).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	request := []byte(`{"ib": 1, "email": "cool@test.com"}`)

	token, err := user.MakeToken("secret", 2)
	assert.NoError(t, err, "An error was not expected")

	first := performJWTRequest(router, "POST", "/email", token, request)

	assert.Equal(t, 200, first.Code, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
