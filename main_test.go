package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
)

// banQuery matches the lookup run by the global Bans middleware
// (middleware.CheckBannedIP). go-sqlmock matches the expected SQL as a regexp.
const banQuery = `SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`

// TestClientIPResolution is a regression guard for how the real router resolves
// c.ClientIP(). In production every request arrives from nginx on loopback with
// Cloudflare's CF-Connecting-IP; the client fully controls X-Forwarded-For and
// X-Real-IP. Each case asserts both the resolved address and that the global ban
// check looked up exactly that address, and that a banned client cannot dodge
// the ban by forging forwarding headers.
func TestClientIPResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupRouter()
	// echo the resolved address; registered after setupRouter so the request
	// still runs through every global middleware, including Bans
	router.GET("/test/clientip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	tests := []struct {
		name       string
		remoteAddr string
		header     map[string]string
		// wantIP is the address the ban check must look up; banned cases each
		// use their own address because banned lookups are cached process-wide
		wantIP string
		banned bool
	}{
		{
			name:       "client ip comes from CF-Connecting-IP",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.1", "X-Forwarded-For": "198.51.100.1"},
			wantIP:     "198.51.100.1",
		},
		{
			name:       "forged X-Forwarded-For is ignored",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.2", "X-Forwarded-For": "203.0.113.9, 198.51.100.2"},
			wantIP:     "198.51.100.2",
		},
		{
			name:       "unparseable X-Forwarded-For does not fall back to loopback",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.3", "X-Forwarded-For": "notanip"},
			wantIP:     "198.51.100.3",
		},
		{
			name:       "forged X-Real-IP is ignored",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.4", "X-Real-IP": "203.0.113.9"},
			wantIP:     "198.51.100.4",
		},
		{
			name:       "ipv6 client",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "2001:db8::5"},
			wantIP:     "2001:db8::5",
		},
		{
			name:       "nginx over ipv6 loopback is trusted",
			remoteAddr: "[::1]:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.6"},
			wantIP:     "198.51.100.6",
		},
		{
			name:       "CF-Connecting-IP from a non-loopback peer is ignored",
			remoteAddr: "192.0.2.7:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.7", "X-Forwarded-For": "198.51.100.7"},
			wantIP:     "192.0.2.7",
		},
		{
			name:       "forwarding headers without CF-Connecting-IP are ignored",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"X-Forwarded-For": "203.0.113.9", "X-Real-IP": "203.0.113.9"},
			wantIP:     "127.0.0.1",
		},
		{
			name:       "unparseable CF-Connecting-IP falls back to the peer",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "notanip", "X-Forwarded-For": "203.0.113.9"},
			wantIP:     "127.0.0.1",
		},
		{
			name:       "banned client forging X-Forwarded-For is denied",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.21", "X-Forwarded-For": "203.0.113.9, 198.51.100.21"},
			wantIP:     "198.51.100.21",
			banned:     true,
		},
		{
			name:       "banned client sending unparseable X-Forwarded-For is denied",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.22", "X-Forwarded-For": "notanip"},
			wantIP:     "198.51.100.22",
			banned:     true,
		},
		{
			name:       "banned client forging X-Real-IP is denied",
			remoteAddr: "127.0.0.1:40000",
			header:     map[string]string{"CF-Connecting-IP": "198.51.100.23", "X-Real-IP": "203.0.113.9"},
			wantIP:     "198.51.100.23",
			banned:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := db.NewTestDb()
			require.NoError(t, err)
			defer db.CloseDb()

			count := 0
			if tt.banned {
				count = 1
			}
			mock.ExpectQuery(banQuery).
				WithArgs(tt.wantIP).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))

			req := httptest.NewRequest(http.MethodGet, "/test/clientip", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.header {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.banned {
				// the 403 alone proves the lookup used wantIP: a lookup of any
				// other address misses the cache and mismatches the mock, which
				// fails with 500. The query itself is skipped on repeat runs
				// (-count>1), because the ban is then served from the cache.
				assert.Equal(t, http.StatusForbidden, w.Code, "banned client must be denied, body: %s", w.Body.String())
				return
			}

			assert.Equal(t, http.StatusOK, w.Code, "unexpected status, body: %s", w.Body.String())
			assert.Equal(t, tt.wantIP, w.Body.String(), "resolved client ip")
			assert.NoError(t, mock.ExpectationsWereMet(), "ban check must look up %s", tt.wantIP)
		})
	}
}
