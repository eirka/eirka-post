package utils

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strconv"

	e "github.com/techjanitor/pram-post/errors"
)

var (
	maxLogins    int  = 5
	limitSeconds uint = 3
)

// will increment a counter in redis to limit login attempts
func LoginCounter(userid uint) (err error) {

	// convert userid to string
	uid := strconv.Itoa(int(userid))

	// Initialize cache handle
	cache := RedisCache

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

// will increment a redis key
func (c *RedisStore) Incr(key string) (result int, err error) {
	conn := c.pool.Get()
	defer conn.Close()

	raw, err := conn.Do("INCR", key)
	if raw == nil {
		return 0, ErrCacheMiss
	}
	result, err = redis.Int(raw, err)
	if err != nil {
		return
	}

	return
}

// will set expire on a redis key
func (c *RedisStore) Expire(key string, timeout uint) (err error) {
	conn := c.pool.Get()
	defer conn.Close()

	_, err = conn.Do("EXPIRE", key, timeout)

	return
}
