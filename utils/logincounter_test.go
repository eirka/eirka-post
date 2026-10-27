package utils

import (
	"testing"
	"time"

	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/redis"
)

// fixClock pins the clock LoginCounter uses to pick a window
func fixClock(t *testing.T, unix int64) {
	orig := now
	now = func() time.Time { return time.Unix(unix, 0) }
	t.Cleanup(func() { now = orig })
}

// window 10 is unix seconds [1800, 1980)
const (
	testIPKey     = "login:ip:10.0.0.1:10"
	testNameIPKey = "login:nameip:bob:10.0.0.1:10"
	testNameKey   = "login:name:bob:10"
)

// mockCounters registers INCR and EXPIRE for each key, INCR returning its count
// in counts, and returns the INCR commands for checking which were run
func mockCounters(counts map[string]int) map[string]*redigomock.Cmd {
	redis.NewRedisMock()

	incrs := make(map[string]*redigomock.Cmd)
	for key, count := range counts {
		incrs[key] = redis.Cache.Mock.Command("INCR", key).Expect(int64(count))
		redis.Cache.Mock.Command("EXPIRE", key, loginWindow).Expect("OK")
	}
	return incrs
}

func TestLoginCounter(t *testing.T) {
	fixClock(t, 1800)

	tests := []struct {
		name   string
		ip     int
		nameIP int
		total  int
		want   error
		// counting stops at the first count over its limit
		wantCounted []string
	}{
		{name: "first attempt", ip: 1, nameIP: 1, total: 1, want: nil, wantCounted: []string{testIPKey, testNameIPKey, testNameKey}},
		{name: "every count at its limit", ip: 20, nameIP: 5, total: 50, want: nil, wantCounted: []string{testIPKey, testNameIPKey, testNameKey}},
		{name: "ip over its limit", ip: 21, nameIP: 1, total: 1, want: e.ErrMaxLogins, wantCounted: []string{testIPKey}},
		{name: "name from this ip over its limit", ip: 6, nameIP: 6, total: 6, want: e.ErrMaxLogins, wantCounted: []string{testIPKey, testNameIPKey}},
		{name: "name over its overall limit", ip: 1, nameIP: 1, total: 51, want: e.ErrMaxLogins, wantCounted: []string{testIPKey, testNameIPKey, testNameKey}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incrs := mockCounters(map[string]int{testIPKey: tt.ip, testNameIPKey: tt.nameIP, testNameKey: tt.total})

			assert.Equal(t, tt.want, LoginCounter("bob", "10.0.0.1"))

			for key, incr := range incrs {
				want := 0
				for _, counted := range tt.wantCounted {
					if counted == key {
						want = 1
					}
				}
				assert.Equal(t, want, redis.Cache.Mock.Stats(incr), "times %s was counted", key)
			}
		})
	}
}

// TestLoginCounterName checks every spelling the users table treats as the same
// name shares its counts
func TestLoginCounterName(t *testing.T) {
	fixClock(t, 1800)

	for _, name := range []string{"bob", "Bob", "BOB", "bob ", "Bob  "} {
		t.Run(name, func(t *testing.T) {
			incrs := mockCounters(map[string]int{testIPKey: 1, testNameIPKey: 1, testNameKey: 1})

			assert.NoError(t, LoginCounter(name, "10.0.0.1"))
			assert.Equal(t, 1, redis.Cache.Mock.Stats(incrs[testNameIPKey]), "%q should count against %s", name, testNameIPKey)
			assert.Equal(t, 1, redis.Cache.Mock.Stats(incrs[testNameKey]), "%q should count against %s", name, testNameKey)
		})
	}
}

// TestLoginCounterIP checks an ipv6 client is counted per /64, and per /48 for
// the attempts its network makes at one name, so moving to other addresses in
// its own allocation does not reset its counts
func TestLoginCounterIP(t *testing.T) {
	fixClock(t, 1800)

	tests := []struct {
		ip      string
		client  string
		network string
	}{
		{ip: "10.0.0.1", client: "10.0.0.1", network: "10.0.0.1"},
		{ip: "::ffff:10.0.0.1", client: "10.0.0.1", network: "10.0.0.1"},
		{ip: "2001:db8:1:2::1", client: "2001:db8:1:2::/64", network: "2001:db8:1::/48"},
		{ip: "2001:db8:1:2:ffff:ffff:ffff:ffff", client: "2001:db8:1:2::/64", network: "2001:db8:1::/48"},
		{ip: "2001:db8:1:ff00::1", client: "2001:db8:1:ff00::/64", network: "2001:db8:1::/48"},
		{ip: "2001:db8:2::1", client: "2001:db8:2::/64", network: "2001:db8:2::/48"},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ipKey := "login:ip:" + tt.client + ":10"
			nameIPKey := "login:nameip:bob:" + tt.network + ":10"
			incrs := mockCounters(map[string]int{ipKey: 1, nameIPKey: 1, testNameKey: 1})

			assert.NoError(t, LoginCounter("bob", tt.ip))
			assert.Equal(t, 1, redis.Cache.Mock.Stats(incrs[ipKey]), "%s should count against %s", tt.ip, ipKey)
			assert.Equal(t, 1, redis.Cache.Mock.Stats(incrs[nameIPKey]), "%s should count against %s", tt.ip, nameIPKey)
		})
	}
}

// TestLoginCounterWindows checks the counts are per fixed window: every attempt
// inside a window uses the same keys, and the next window starts fresh keys
func TestLoginCounterWindows(t *testing.T) {
	tests := []struct {
		unix   int64
		window string
	}{
		{unix: 1800, window: "10"},
		{unix: 1979, window: "10"},
		{unix: 1980, window: "11"},
	}

	for _, tt := range tests {
		t.Run(time.Unix(tt.unix, 0).UTC().Format(time.TimeOnly), func(t *testing.T) {
			fixClock(t, tt.unix)

			keys := []string{
				"login:ip:10.0.0.1:" + tt.window,
				"login:nameip:bob:10.0.0.1:" + tt.window,
				"login:name:bob:" + tt.window,
			}
			incrs := mockCounters(map[string]int{keys[0]: 1, keys[1]: 1, keys[2]: 1})

			assert.NoError(t, LoginCounter("bob", "10.0.0.1"))
			for _, key := range keys {
				assert.Equal(t, 1, redis.Cache.Mock.Stats(incrs[key]), "times %s was counted", key)
			}
		})
	}
}

func TestLoginCounterInvalidInputs(t *testing.T) {
	fixClock(t, 1800)

	for _, tt := range []struct{ name, login, ip string }{
		{name: "empty name", login: "", ip: "10.0.0.1"},
		{name: "empty ip", login: "bob", ip: ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// redis would accept any key, so only the input check can fail this
			redis.NewRedisMock()
			incr := redis.Cache.Mock.Command("INCR", redigomock.NewAnyData()).Expect(int64(1))
			redis.Cache.Mock.Command("EXPIRE", redigomock.NewAnyData(), loginWindow).Expect("OK")

			assert.Equal(t, e.ErrInternalError, LoginCounter(tt.login, tt.ip))
			assert.Equal(t, 0, redis.Cache.Mock.Stats(incr), "nothing is counted for an invalid input")
		})
	}
}

func TestLoginCounterRedisErrors(t *testing.T) {
	fixClock(t, 1800)

	for _, failKey := range []string{testIPKey, testNameIPKey, testNameKey} {
		for _, failOn := range []string{"INCR", "EXPIRE"} {
			t.Run(failOn+" "+failKey, func(t *testing.T) {
				redis.NewRedisMock()

				for _, key := range []string{testIPKey, testNameIPKey, testNameKey} {
					incr := redis.Cache.Mock.Command("INCR", key)
					expire := redis.Cache.Mock.Command("EXPIRE", key, loginWindow)
					switch {
					case key == failKey && failOn == "INCR":
						incr.ExpectError(redis.ErrCacheMiss)
					case key == failKey && failOn == "EXPIRE":
						incr.Expect(int64(1))
						expire.ExpectError(redis.ErrCacheMiss)
					default:
						incr.Expect(int64(1))
						expire.Expect("OK")
					}
				}

				assert.Equal(t, e.ErrInternalError, LoginCounter("bob", "10.0.0.1"))
			})
		}
	}
}
