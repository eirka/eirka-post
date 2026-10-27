package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/csrf"
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

// countingReader records how much of a request body was consumed
type countingReader struct {
	r    io.Reader
	read int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += n
	return n, err
}

// csrfPair returns a csrf cookie and the matching masked token, as issued by
// csrf.Cookie on eirka-index
func csrfPair(t *testing.T) (*http.Cookie, string) {
	r := gin.New()
	var token string
	r.GET("/", csrf.Cookie(), func(c *gin.Context) {
		token = c.GetString("csrf_token")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	require.NotEmpty(t, token)
	return cookies[0], token
}

// TestBodyLimitBeforeCSRF is a regression guard for M1: the real router must
// cap a request body before csrf.Verify reads it. Without an X-XSRF-TOKEN
// header, Verify reads the token from the form, which parses the whole body,
// and it runs on every route (including NoRoute) before any auth.
func TestBodyLimitBeforeCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orig := config.Settings.Limits.ImageMaxSize
	config.Settings.Limits.ImageMaxSize = 100 << 10
	defer func() { config.Settings.Limits.ImageMaxSize = orig }()

	const (
		smallMax  = 64 << 10
		uploadMax = 100<<10 + 1<<20
		// far past either limit; before the fix it was read in full
		bodySize = 4 << 20
	)

	multipartHead := "--x\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.jpg\"\r\nContent-Type: image/jpeg\r\n\r\n"

	tests := []struct {
		name        string
		path        string
		contentType string
		head        string
		// size of the body after head; bodySize unless set
		size int
		// declared sends the true Content-Length; otherwise the body is chunked
		declared bool
		wantCode int
		maxRead  int
		// fullRead is whether the whole body got past the limit
		fullRead bool
	}{
		{name: "chunked multipart upload without csrf header", path: "/thread/new", contentType: "multipart/form-data; boundary=x", head: multipartHead, wantCode: http.StatusForbidden, maxRead: uploadMax + 1},
		{name: "chunked multipart to a json route", path: "/login", contentType: "multipart/form-data; boundary=x", head: multipartHead, wantCode: http.StatusForbidden, maxRead: smallMax + 1},
		{name: "chunked multipart to an unknown route", path: "/nope", contentType: "multipart/form-data; boundary=x", head: multipartHead, wantCode: http.StatusForbidden, maxRead: smallMax + 1},
		{name: "chunked urlencoded form", path: "/register", contentType: "application/x-www-form-urlencoded", head: "csrf_token=", wantCode: http.StatusForbidden, maxRead: smallMax + 1},
		{name: "declared oversized multipart", path: "/thread/reply", contentType: "multipart/form-data; boundary=x", head: multipartHead, declared: true, wantCode: http.StatusRequestEntityTooLarge, maxRead: 0},
		{name: "declared oversized json", path: "/login", contentType: "application/json", head: `{"name":"`, declared: true, wantCode: http.StatusRequestEntityTooLarge, maxRead: 0},
		{name: "declared upload-sized multipart to a json route", path: "/login", contentType: "multipart/form-data; boundary=x", head: multipartHead, size: smallMax * 4, declared: true, wantCode: http.StatusRequestEntityTooLarge, maxRead: 0},
		// each upload route takes an upload-sized body; csrf.Verify reads all
		// of it looking for the token, and rejects the request without one
		{name: "declared upload-sized multipart to /thread/new", path: "/thread/new", contentType: "multipart/form-data; boundary=x", head: multipartHead, size: smallMax * 4, declared: true, wantCode: http.StatusForbidden, maxRead: uploadMax, fullRead: true},
		{name: "declared upload-sized multipart to /thread/reply", path: "/thread/reply", contentType: "multipart/form-data; boundary=x", head: multipartHead, size: smallMax * 4, declared: true, wantCode: http.StatusForbidden, maxRead: uploadMax, fullRead: true},
		{name: "declared upload-sized multipart to /user/avatar", path: "/user/avatar", contentType: "multipart/form-data; boundary=x", head: multipartHead, size: smallMax * 4, declared: true, wantCode: http.StatusForbidden, maxRead: uploadMax, fullRead: true},
	}

	router := setupRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := tt.size
			if size == 0 {
				size = bodySize
			}
			body := &countingReader{r: io.MultiReader(strings.NewReader(tt.head), bytes.NewReader(make([]byte, size)))}
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.Header.Set("Content-Type", tt.contentType)
			req.ContentLength = -1
			if tt.declared {
				req.ContentLength = int64(len(tt.head) + size)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code, "body: %s", w.Body.String())
			assert.LessOrEqual(t, body.read, tt.maxRead, "body must not be read past the limit")
			if tt.fullRead {
				assert.Equal(t, len(tt.head)+size, body.read, "an upload route must take an upload-sized body")
			}
		})
	}
}

// TestBodyLimitReachesHandler checks a body within the small limit reaches a
// handler on the real router in full, and one past it is cut off, multipart
// included, on a route that is not an upload route. The csrf token is sent in
// the header, so Verify leaves the body alone.
func TestBodyLimitReachesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupRouter()
	// read the whole body; registered after setupRouter so the request still
	// runs through every global middleware
	router.POST("/test/body", func(c *gin.Context) {
		n, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			c.String(http.StatusRequestEntityTooLarge, "%d", n)
			return
		}
		c.String(http.StatusOK, "%d", n)
	})

	cookie, token := csrfPair(t)

	tests := []struct {
		name        string
		contentType string
		size        int
		wantCode    int
	}{
		{name: "json body at the small limit", contentType: "application/json", size: 64 << 10, wantCode: http.StatusOK},
		{name: "chunked json body over the small limit", contentType: "application/json", size: 64<<10 + 1, wantCode: http.StatusRequestEntityTooLarge},
		{name: "chunked multipart body over the small limit", contentType: "multipart/form-data; boundary=x", size: 64<<10 + 1, wantCode: http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := db.NewTestDb()
			require.NoError(t, err)
			defer db.CloseDb()

			mock.ExpectQuery(banQuery).
				WithArgs("192.0.2.1").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			req := httptest.NewRequest(http.MethodPost, "/test/body", bytes.NewReader(make([]byte, tt.size)))
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set(csrf.HeaderName, token)
			req.AddCookie(cookie)
			req.ContentLength = -1

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code, "body: %s", w.Body.String())
			if tt.wantCode == http.StatusOK {
				assert.Equal(t, strconv.Itoa(tt.size), w.Body.String(), "the whole body should reach the handler")
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
