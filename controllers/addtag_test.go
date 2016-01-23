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

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(user.Auth(false))

	router.POST("/tag/add", AddTagController)

	first := performRequest(router, "POST", "/tag/add")

	assert.Equal(t, first.Code, 400, "HTTP request code should match")

	request1 := []byte(`{"ib": 1, "tag": 1, "image": 1}`)

	second := performJsonRequest(router, "POST", "/tag/add", request1)

	assert.Equal(t, second.Code, 200, "HTTP request code should match")
	assert.JSONEq(t, second.Body.String(), successMessage(audit.AuditAddTag), "HTTP response should match")

}
