package controllers

import (
	"testing"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/db"
	//e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
)

func TestEmailController(t *testing.T) {

	var err error

	user.Secret = "secret"

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(user.Auth(true))

	router.POST("/email", EmailController)

	user := user.DefaultUser()
	user.SetID(2)
	user.SetAuthenticated()

	//	hash, err := user.HashPassword("testpass")
	//	if assert.NoError(t, err, "An error was not expected") {
	//		assert.NotEmpty(t, hash, "token should be returned")
	//	}

	token, err := user.CreateToken()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, token, "token should be returned")
	}

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	rows := sqlmock.NewRows([]string{"name", "email"}).AddRow("test", "old@test.com")
	mock.ExpectQuery(`SELECT user_name,user_email FROM users WHERE user_id`).WillReturnRows(rows)

	mock.ExpectExec("UPDATE users SET user_email").
		WithArgs("cool@test.com", 2).
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := []byte(`{"ib": 1, "email": "test@test.com"}`)

	first := performJwtJSONRequest(router, "POST", "/email", token, request)

	assert.Equal(t, first.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
