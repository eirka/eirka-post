package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
)

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBans(t *testing.T) {

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// posts need to be verified
	router.Use(Bans())

	router.POST("/reply", func(c *gin.Context) {
		c.String(200, "OK")
		return
	})

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")

	bannedrow := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip`).WillReturnRows(bannedrow)

	first := performRequest(router, "POST", "/reply")

	assert.Equal(t, first.Code, 401, "HTTP request code should match")

	clearrow := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip`).WillReturnRows(clearrow)

	second := performRequest(router, "POST", "/reply")

	assert.Equal(t, second.Code, 200, "HTTP request code should match")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}
