package utils

import (
	"fmt"
	"net/netip"
	"strings"
	"time"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
)

var (
	// loginWindow is the length of a rate limit window in seconds
	loginWindow uint = 180
	// maxIPLogins is how many attempts one ip may make per window, for any names
	maxIPLogins = 20
	// maxNameIPLogins is how many attempts one network may make per window at
	// one name
	maxNameIPLogins = 5
	// maxNameLogins is how many attempts one name may receive per window, from
	// all ips together. It is well above maxNameIPLogins, so locking a name out
	// of login takes about ten networks rather than one
	maxNameLogins = 50

	// clientPrefix and networkPrefix group ipv6 addresses for counting. A
	// client is normally given a whole /64 and can use any address in it. A
	// network is a /48: one site's allocation, and free from tunnel brokers,
	// so anything narrower lets one attacker pose as many networks. Carriers
	// can put many devices in one /48, so only the per-name count uses it
	clientPrefix  = 64
	networkPrefix = 48

	// now picks the current window; a variable so tests can fix the clock
	now = time.Now
)

// LoginCounter counts a login attempt and returns ErrMaxLogins once the client
// ip, the requested name from the client's network, or the requested name
// overall is over its limit for the current window. It runs before the name is
// looked up, so names that do not exist are limited the same as real accounts.
//
// The counts are checked in that order and stop at the first one over its
// limit, so a network cannot add more than maxNameIPLogins to a name's overall
// count per window. For ipv4 the client and its network are the same address.
//
// Windows are fixed: the window number is part of the key, so a steady trickle
// of attempts, such as the logins from a busy shared ip, cannot keep a count
// alive the way refreshing a TTL on every attempt would.
func LoginCounter(name, ip string) error {
	// Validate inputs
	if name == "" || ip == "" {
		return e.ErrInternalError
	}

	window := now().Unix() / int64(loginWindow)
	client := loginIP(ip, clientPrefix)
	network := loginIP(ip, networkPrefix)

	// the users table matches names case insensitively and ignores trailing
	// spaces, so every spelling of a name shares a count
	name = strings.ToLower(strings.TrimSpace(name))

	// keys are like login:ip:10.0.0.1:9876543,
	// login:nameip:bob:10.0.0.1:9876543 and login:name:bob:9876543
	counters := []struct {
		key string
		max int
	}{
		{key: fmt.Sprintf("login:ip:%s:%d", client, window), max: maxIPLogins},
		{key: fmt.Sprintf("login:nameip:%s:%s:%d", name, network, window), max: maxNameIPLogins},
		{key: fmt.Sprintf("login:name:%s:%d", name, window), max: maxNameLogins},
	}

	for _, counter := range counters {
		err := countLogin(counter.key, counter.max)
		if err != nil {
			return err
		}
	}

	return nil
}

// loginIP is the source an attempt is counted against: the address for ipv4,
// including ipv4-mapped ipv6, and its /bits prefix for ipv6
func loginIP(ip string, bits int) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	// only fails for a prefix length out of range
	prefix, _ := addr.Prefix(bits)
	return prefix.String()
}

// countLogin increments the counter at key and returns ErrMaxLogins once it is
// over max, or ErrInternalError if redis fails
func countLogin(key string, max int) error {
	// increment login key
	result, err := redis.Cache.Incr(key)
	if err != nil {
		return e.ErrInternalError
	}

	// set expire on key; it is set on every attempt, so a failure here is
	// retried by the next one, and an old window's key is never read again
	err = redis.Cache.Expire(key, loginWindow)
	if err != nil {
		return e.ErrInternalError
	}

	// return error if we are over the max
	if result > max {
		return e.ErrMaxLogins
	}

	return nil
}
