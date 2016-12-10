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

	// key is like login:10.0.0.1:21
	key := fmt.Sprintf("login:%s:%d", ip, userid)

	// increment login key
	result, err := redis.Cache.Incr(key)
	if err != nil {
		return e.ErrInternalError
	}

	// set expire ib key
	err = redis.Cache.Expire(key, limitSeconds)
	if err != nil {
		return e.ErrInternalError
	}

	// return error if we are at the max
	if result >= maxLogins {
		return e.ErrMaxLogins
	}

	return

}
