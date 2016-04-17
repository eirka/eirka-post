package utils

import (
	"testing"

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
