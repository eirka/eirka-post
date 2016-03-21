package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"

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
	user.SetId(2)
	user.SetAuthenticated()

	user.hash, err = user.HashPassword("testpassword")
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotNil(t, user.hash, "password should be returned")
	}

	assert.True(t, user.ComparePassword("testpassword"), "Password should validate")

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

	first := performJwtJsonRequest(router, "POST", "/email", token, request)

	assert.Equal(t, first.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
