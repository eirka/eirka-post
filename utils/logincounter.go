package utils

import (
	"fmt"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
)

var (
	maxLogins         = 5
	limitSeconds uint = 180
)

// LoginCounter will increment a counter in redis to limit login attempts
func LoginCounter(userid uint, ip string) (err error) {

	// Initialize cache handle
	cache := redis.Cache

	// key is like login:21
	key := fmt.Sprintf("login:%s:%d", ip, userid)

	// increment login key
	result, err := cache.Incr(key)
	if err != nil {
		return e.ErrInternalError
	}

	// increment login key
	err = cache.Expire(key, limitSeconds)
	if err != nil {
		return e.ErrInternalError
	}

	if result >= maxLogins {
		return e.ErrMaxLogins
	}

	return

}
