package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"strconv"

	e "github.com/techjanitor/pram-post/errors"
)

var nRandBytes = 32

// creates a new random session id with user id
func NewSession(userid uint) (cookieToken string, err error) {

	uid := strconv.Itoa(int(userid))

	// Initialize cache handle
	cache := RedisCache

	// make slice for token, with user + semicolon
	token := make([]byte, nRandBytes+len(uid)+1)
	// copy key into token
	copy(token, []byte(uid))
	// add semicolon
	token[len(uid)] = ';'

	// read in random bytes
	_, err = rand.Read(token[len(uid)+1:])
	if err != nil {
		return
	}

	// goes in the cookie
	cookieToken = base64.URLEncoding.EncodeToString(token)

	// goes to redis
	sum := md5.Sum(token)
	storageToken := base64.StdEncoding.EncodeToString(sum[:])

	// set key in redis
	err = cache.SetEx(storageToken, 2592000, []byte(uid))
	if err != nil {
		return
	}

	return

}

// validate compares provided session id to redis
func ValidateSession(key []byte) (err error) {

	// Initialize cache handle
	cache := RedisCache

	// decode key
	token, err := base64.URLEncoding.DecodeString(string(key))
	if err != nil {
		return
	}

	// get uid
	index := bytes.IndexByte(token, ';')

	// check to see if user is there
	if index < 0 {
		return e.ErrInvalidSession
	}

	// get given uid
	uid := string(token[:index])

	// hash token
	sum := md5.Sum(token)

	// base64 encode sum
	providedHash := base64.StdEncoding.EncodeToString(sum[:])

	// check for match
	result, err := cache.Get(providedHash)
	if err == ErrCacheMiss {
		return e.ErrInvalidSession
	}
	if err != nil {
		return
	}

	// check if uid matches
	if uid != string(result) {
		return e.ErrInvalidSession
	}

	return

}