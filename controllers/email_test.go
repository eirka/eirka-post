package controllers

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eirka/eirka-libs/audit"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"
)

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

	request1 := []byte(`{"ib": 1, "email": "test@test.com"}`)

	second := performJwtJsonRequest(router, "POST", "/email", token, request1)

	assert.Equal(t, second.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, second.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

	request2 := []byte(`{"ib": 1, "email": "test@test.com"}`)

	third := performJwtJsonRequest(router, "POST", "/email", token, request2)

	assert.Equal(t, third.Code, 400, "HTTP request code should match")
	assert.JSONEq(t, third.Body.String(), errorMessage(e.ErrEmailSame), "HTTP response should match")

	request3 := []byte(`{"ib": 1, "email": "test@cool.com"}`)

	fourth := performJwtJsonRequest(router, "POST", "/email", token, request3)

	assert.Equal(t, fourth.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, fourth.Body.String(), successMessage(audit.AuditEmailUpdate), "HTTP response should match")

}

func TestEmailControllerBadRequests(t *testing.T) {

	var err error

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(user.Auth(true))

	router.POST("/email", EmailController)

	u := user.DefaultUser()
	u.SetId(2)
	u.SetAuthenticated()
	u.Password()

	assert.True(t, u.ComparePassword("testpassword"), "Test user password should be set")

	token, err := u.CreateToken()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, token, "token should be returned")
	}

	request1 := []byte(`{"ib": 1}`)

	first := performJwtJsonRequest(router, "POST", "/email", token, request1)

	assert.Equal(t, first.Code, 400, "HTTP request code should match")
	assert.JSONEq(t, first.Body.String(), errorMessage(e.ErrInvalidParam), "HTTP response should match")

	request2 := []byte(`{"email": "test@cool.com"}`)

	second := performJwtJsonRequest(router, "POST", "/email", token, request2)

	assert.Equal(t, second.Code, 400, "HTTP request code should match")
	assert.JSONEq(t, second.Body.String(), errorMessage(e.ErrInvalidParam), "HTTP response should match")

}
