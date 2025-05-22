package utils

import (
	"testing"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

func TestLoginCounter(t *testing.T) {
	redis.NewRedisMock()

	redis.Cache.Mock.Command("INCR", "login:10.0.0.1:2").Expect([]byte("1"))
	redis.Cache.Mock.Command("EXPIRE", "login:10.0.0.1:2", redigomock.NewAnyData())

	err := LoginCounter(2, "10.0.0.1")

	assert.NoError(t, err, "An error was not expected")
}

func TestLoginCounterMax(t *testing.T) {
	redis.NewRedisMock()

	redis.Cache.Mock.Command("INCR", "login:10.0.0.1:2").Expect([]byte("5"))
	redis.Cache.Mock.Command("EXPIRE", "login:10.0.0.1:2", redigomock.NewAnyData())

	err := LoginCounter(2, "10.0.0.1")

	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrMaxLogins, err, "Error should match")
	}
}

func TestLoginCounterInvalidInputs(t *testing.T) {
	// Test with zero user ID
	err := LoginCounter(0, "10.0.0.1")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInternalError, err, "Error should match")
	}

	// Test with empty IP
	err = LoginCounter(2, "")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInternalError, err, "Error should match")
	}
}

func TestLoginCounterRedisErrors(t *testing.T) {
	// Test Incr failure
	redis.NewRedisMock()
	redis.Cache.Mock.Command("INCR", "login:10.0.0.1:2").ExpectError(redis.ErrCacheMiss)

	err := LoginCounter(2, "10.0.0.1")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInternalError, err, "Error should match")
	}

	// Test Expire failure
	redis.NewRedisMock()
	redis.Cache.Mock.Command("INCR", "login:10.0.0.1:2").Expect([]byte("1"))
	redis.Cache.Mock.Command("EXPIRE", "login:10.0.0.1:2", redigomock.NewAnyData()).ExpectError(redis.ErrCacheMiss)

	err = LoginCounter(2, "10.0.0.1")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrInternalError, err, "Error should match")
	}
}
