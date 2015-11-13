package utils

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strconv"

	e "github.com/techjanitor/pram-libs/errors"
	"github.com/techjanitor/pram-libs/redis"
)

var (
	maxLogins    int  = 5
	limitSeconds uint = 300
)

// will increment a counter in redis to limit login attempts
func LoginCounter(userid uint) (err error) {

	// convert userid to string
	uid := strconv.Itoa(int(userid))

	// Initialize cache handle
	cache := redis.RedisCache

	// key is like login:21
	key := fmt.Sprintf("login:%s", uid)

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
