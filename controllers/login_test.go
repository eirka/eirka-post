package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/user"
)

// fromNameQuery matches the account lookup in user.FromName
const fromNameQuery = `select user_id, user_password from users where user_name = \?`

func TestLoginControllerValidation(t *testing.T) {
	// Set up the gin context
	gin.SetMode(gin.ReleaseMode)

	// Create a router
	router := gin.New()
	router.TrustedPlatform = "X-Real-IP"
	router.POST("/login", LoginController)

	// Test cases for invalid parameters
	testCases := []struct {
		name string
		body string
		code int
	}{
		{"missing_name", `{"ib":1,"password":"test"}`, http.StatusBadRequest},
		{"missing_password", `{"ib":1,"name":"test"}`, http.StatusBadRequest},
		{"missing_ib", `{"name":"test","password":"test"}`, http.StatusBadRequest},
		{"invalid_json", `{invalid_json}`, http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest("POST", "/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", "127.0.0.1")

			// Create a response recorder
			w := httptest.NewRecorder()

			// Perform the request
			router.ServeHTTP(w, req)

			// Check response
			assert.Equal(t, tc.code, w.Code)
			assert.Contains(t, w.Body.String(), "error_message")
		})
	}
}

// loginRedis mocks the login counters. INCR fails with err, or returns the
// count in counts for the counter's kind (the word after "login:"), and 1 for a
// kind not in counts. It returns the keys counted, in order.
func loginRedis(counts map[string]int64, err error) *[]string {
	redis.NewRedisMock()

	keys := &[]string{}
	redis.Cache.Mock.Command("INCR", redigomock.NewAnyData()).Handle(func(args []interface{}) (interface{}, error) {
		key := args[0].(string)
		*keys = append(*keys, key)
		if err != nil {
			return nil, err
		}
		if count, ok := counts[strings.Split(key, ":")[1]]; ok {
			return count, nil
		}
		return int64(1), nil
	})
	redis.Cache.Mock.Command("EXPIRE", redigomock.NewAnyData(), redigomock.NewAnyData()).Expect("OK")

	return keys
}

// postLogin sends a login request to LoginController
func postLogin(name, password string) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/login", LoginController)

	body, _ := json.Marshal(map[string]any{"ib": 1, "name": name, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestLoginController(t *testing.T) {
	gin.SetMode(gin.TestMode)

	origSecret := config.Settings.Session.NewSecret
	config.Settings.Session.NewSecret = "0123456789abcdef0123456789abcdef"
	defer func() { config.Settings.Session.NewSecret = origSecret }()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.MinCost)
	require.NoError(t, err)

	account := func(uid int) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"user_id", "user_password"}).AddRow(uid, hash)
	}

	tests := []struct {
		name     string
		login    string
		password string
		// counts for the counters over their limit, by kind; every other
		// counter returns 1
		counts   map[string]int64
		redisErr error
		// noRedis leaves redis unmocked, so any counting fails the request
		noRedis bool
		// rows answers the account lookup; it is not expected when both are nil
		rows        *sqlmock.Rows
		dbErr       error
		wantCode    int
		wantMessage string
		wantCookie  bool
	}{
		{name: "correct password", login: "bob", password: "correct horse", rows: account(2), wantCode: http.StatusOK, wantMessage: "Login successful", wantCookie: true},
		{name: "wrong password", login: "bob", password: "wrong horse", rows: account(2), wantCode: http.StatusBadRequest, wantMessage: errInvalidLogin.Error()},
		{name: "unknown name", login: "nobody", password: "correct horse", rows: sqlmock.NewRows([]string{"user_id", "user_password"}), wantCode: http.StatusBadRequest, wantMessage: errInvalidLogin.Error()},
		{name: "invalid account id", login: "ghost", password: "correct horse", rows: account(0), wantCode: http.StatusBadRequest, wantMessage: errInvalidLogin.Error()},
		{name: "database error", login: "bob", password: "correct horse", dbErr: errors.New("connection refused"), wantCode: http.StatusInternalServerError, wantMessage: e.ErrInternalError.Error()},
		// every limit is checked before the lookup, so a correct password is
		// refused too, and an unknown name gets the same answer
		{name: "ip over its limit", login: "bob", password: "correct horse", counts: map[string]int64{"ip": 1000}, wantCode: http.StatusTooManyRequests, wantMessage: e.ErrMaxLogins.Error()},
		{name: "name from this ip over its limit", login: "bob", password: "correct horse", counts: map[string]int64{"nameip": 1000}, wantCode: http.StatusTooManyRequests, wantMessage: e.ErrMaxLogins.Error()},
		{name: "name over its overall limit", login: "bob", password: "correct horse", counts: map[string]int64{"name": 1000}, wantCode: http.StatusTooManyRequests, wantMessage: e.ErrMaxLogins.Error()},
		{name: "unknown name over its limit", login: "nobody", password: "correct horse", counts: map[string]int64{"name": 1000}, wantCode: http.StatusTooManyRequests, wantMessage: e.ErrMaxLogins.Error()},
		{name: "redis error", login: "bob", password: "correct horse", redisErr: redis.ErrCacheMiss, wantCode: http.StatusInternalServerError, wantMessage: e.ErrInternalError.Error()},
		{name: "reserved name is rejected before counting", login: "admin", password: "correct horse", noRedis: true, wantCode: http.StatusBadRequest, wantMessage: e.ErrUserNotAllowed.Error()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// with no commands registered, any redis call fails the request
			redis.NewRedisMock()
			if !tt.noRedis {
				loginRedis(tt.counts, tt.redisErr)
			}

			mock, err := db.NewTestDb()
			require.NoError(t, err)
			defer db.CloseDb()

			// with no query expected, a lookup fails the request with a 500
			if tt.rows != nil {
				mock.ExpectQuery(fromNameQuery).WithArgs(tt.login).WillReturnRows(tt.rows)
			} else if tt.dbErr != nil {
				mock.ExpectQuery(fromNameQuery).WithArgs(tt.login).WillReturnError(tt.dbErr)
			}

			w := postLogin(tt.login, tt.password)

			assert.Equal(t, tt.wantCode, w.Code, "body: %s", w.Body.String())
			assert.Contains(t, w.Body.String(), tt.wantMessage)
			assert.NoError(t, mock.ExpectationsWereMet())

			var jwt *http.Cookie
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == user.CookieName {
					jwt = cookie
				}
			}
			if tt.wantCookie {
				require.NotNil(t, jwt, "a successful login sets the session cookie")
				assert.NotEmpty(t, jwt.Value)
			} else {
				assert.Nil(t, jwt, "a failed login must not set the session cookie")
			}
		})
	}
}

// TestLoginControllerCounts checks the controller counts the attempt against
// the client ip, the name from that ip, and the name overall, in that order,
// with the name in the form every spelling of it shares
func TestLoginControllerCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	keys := loginRedis(nil, nil)

	mock, err := db.NewTestDb()
	require.NoError(t, err)
	defer db.CloseDb()
	mock.ExpectQuery(fromNameQuery).WillReturnRows(sqlmock.NewRows([]string{"user_id", "user_password"}))

	w := postLogin("Bob", "wrong horse")
	assert.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())

	// the trailing number is the window
	want := []string{
		`^login:ip:192\.0\.2\.1:\d+$`,
		`^login:nameip:bob:192\.0\.2\.1:\d+$`,
		`^login:name:bob:\d+$`,
	}
	if assert.Len(t, *keys, len(want), "keys counted: %v", *keys) {
		for i, pattern := range want {
			assert.Regexp(t, pattern, (*keys)[i])
		}
	}
}

// TestLoginUnknownNameTiming checks that a login for an unknown name gives the
// same answer as a wrong password and takes about as long, so neither reveals
// which names are registered. Without the dummy comparison an unknown name
// returns in well under a millisecond, against tens of milliseconds for a
// bcrypt comparison at the default cost, so the bound here is loose.
func TestLoginUnknownNameTiming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hash, err := bcrypt.GenerateFromPassword([]byte("correct horse"), bcrypt.DefaultCost)
	require.NoError(t, err)

	// generate the dummy hash now, so its one-off cost is not timed
	dummyHash()

	attempt := func(rows *sqlmock.Rows) (time.Duration, *httptest.ResponseRecorder) {
		loginRedis(nil, nil)

		mock, err := db.NewTestDb()
		require.NoError(t, err)
		defer db.CloseDb()
		mock.ExpectQuery(fromNameQuery).WillReturnRows(rows)

		start := time.Now()
		w := postLogin("bob", "wrong horse")
		elapsed := time.Since(start)

		assert.NoError(t, mock.ExpectationsWereMet())
		return elapsed, w
	}

	// the fastest of a few runs each, to keep scheduler noise out
	unknown, wrong := time.Duration(1<<62), time.Duration(1<<62)
	for range 3 {
		d, unknownResp := attempt(sqlmock.NewRows([]string{"user_id", "user_password"}))
		unknown = min(unknown, d)

		d, wrongResp := attempt(sqlmock.NewRows([]string{"user_id", "user_password"}).AddRow(2, hash))
		wrong = min(wrong, d)

		assert.Equal(t, wrongResp.Code, unknownResp.Code, "same status for an unknown name and a wrong password")
		assert.Equal(t, wrongResp.Body.String(), unknownResp.Body.String(), "same body for an unknown name and a wrong password")
	}

	assert.Greater(t, unknown, wrong/2, "an unknown name (%v) should cost about as much as a wrong password (%v)", unknown, wrong)
}
