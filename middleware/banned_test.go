package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/db"
)

func TestBansDirectly(t *testing.T) {
	// This test directly tests the CheckBannedIP function
	bannedIPCache = make(map[string]time.Time)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Test a banned IP
	bannedRow := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`).
		WithArgs("192.168.1.1").
		WillReturnRows(bannedRow)

	isBanned, err := CheckBannedIP("192.168.1.1")
	assert.NoError(t, err, "Should not return error for valid IP")
	assert.True(t, isBanned, "IP should be reported as banned")

	// Check the cache works for the same IP
	isBanned, err = CheckBannedIP("192.168.1.1")
	assert.NoError(t, err, "Should not return error for cached IP")
	assert.True(t, isBanned, "Cached IP should be reported as banned")

	// Test a non-banned IP
	clearRow := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`).
		WithArgs("192.168.1.2").
		WillReturnRows(clearRow)

	isBanned, err = CheckBannedIP("192.168.1.2")
	assert.NoError(t, err, "Should not return error for valid IP")
	assert.False(t, isBanned, "IP should not be reported as banned")

	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestBansWithDatabaseError(t *testing.T) {
	// Clear the cache before testing
	bannedIPCache = make(map[string]time.Time)

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Mock a database error
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`).
		WithArgs("192.168.1.3").
		WillReturnError(errors.New("database error"))

	// Direct test of the function
	isBanned, err := CheckBannedIP("192.168.1.3")
	assert.Error(t, err, "Should return error on database failure")
	assert.False(t, isBanned, "Should not report as banned on database error")
	assert.Contains(t, err.Error(), "database error", "Error should be passed through")

	assert.NoError(t, mock.ExpectationsWereMet(), "All database expectations should be met")
}

func TestCheckBannedIP(t *testing.T) {
	// Clear the cache before testing
	bannedIPCache = make(map[string]time.Time)

	// Test invalid IP
	isBanned, err := CheckBannedIP("not-an-ip")
	assert.Error(t, err, "Should return error for invalid IP")
	assert.False(t, isBanned, "Invalid IP should not be reported as banned")
	assert.Contains(t, err.Error(), "invalid IP address format", "Error message should indicate invalid IP format")

	// Test empty IP
	isBanned, err = CheckBannedIP("")
	assert.Error(t, err, "Should return error for empty IP")
	assert.False(t, isBanned, "Empty IP should not be reported as banned")

	// Setup mock for valid IPs
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// Test IPv4
	bannedRow := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`).
		WithArgs("192.168.1.1").
		WillReturnRows(bannedRow)

	isBanned, err = CheckBannedIP("192.168.1.1")
	assert.NoError(t, err, "Should not return error for valid IPv4")
	assert.True(t, isBanned, "Valid banned IPv4 should be reported as banned")

	// Test IPv6
	clearRow := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip = \?`).
		WithArgs("2001:0db8:85a3:0000:0000:8a2e:0370:7334").
		WillReturnRows(clearRow)

	isBanned, err = CheckBannedIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	assert.NoError(t, err, "Should not return error for valid IPv6")
	assert.False(t, isBanned, "Non-banned IPv6 should not be reported as banned")

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}

func TestBansCaching(t *testing.T) {
	// Clear the cache before testing
	bannedIPCache = make(map[string]time.Time)

	// Setup mock
	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	// First call should hit the database
	bannedRow := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_ips WHERE ban_ip`).WillReturnRows(bannedRow)

	isBanned, err := CheckBannedIP("192.168.1.2")
	assert.NoError(t, err, "Should not return error for valid IP")
	assert.True(t, isBanned, "IP should be reported as banned")

	// Second call should use the cache
	isBanned, err = CheckBannedIP("192.168.1.2")
	assert.NoError(t, err, "Should not return error for valid IP")
	assert.True(t, isBanned, "IP should be reported as banned from cache")

	// Database expectations should be met (only one query)
	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")
}
