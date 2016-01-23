package controllers

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	"github.com/eirka/eirka-libs/user"

	local "github.com/eirka/eirka-post/config"
)

func init() {

	// Database connection settings
	dbase := db.Database{

		User:           local.Settings.Database.User,
		Password:       local.Settings.Database.Password,
		Proto:          local.Settings.Database.Proto,
		Host:           local.Settings.Database.Host,
		Database:       local.Settings.Database.Database,
		MaxIdle:        local.Settings.Database.MaxIdle,
		MaxConnections: local.Settings.Database.MaxConnections,
	}

	// Set up DB connection
	dbase.NewDb()

	// Get limits and stuff from database
	config.GetDatabaseSettings()

	user.Secret = "secret"
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performJwtJsonRequest(r http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	fmt.Println(req)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func performJwtFormRequest(r http.Handler, method, path, token string, body bytes.Buffer) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, &body)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestEmailController(t *testing.T) {

	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(user.Auth(true))

	router.POST("/email", EmailController)

	first := performRequest(router, "POST", "/email")

	assert.Equal(t, first.Code, 401, "HTTP request code should match")

	u := user.DefaultUser()
	u.SetId(2)
	u.SetAuthenticated()
	u.Password()

	assert.True(t, u.ComparePassword("testpassword"), "Test user password should be set")

	token, err := u.CreateToken()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, token, "token should be returned")
	}

	request := []byte(`{"ib":"1","email":"test@test.com"}`)

	second := performJwtJsonRequest(router, "POST", "/email", token, request)

	fmt.Println(second)

	assert.Equal(t, second.Code, 200, "HTTP request code should match")

}
