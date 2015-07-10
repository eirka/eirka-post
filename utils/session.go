package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"

	e "github.com/techjanitor/pram-post/errors"
)

var nRandBytes = 32

// creates a new random session id with user id
func NewSession(userid, groupid uint) (cookieToken string, err error) {

	// convert userid to string
	uid := strconv.Itoa(int(userid))
	gid := strconv.Itoa(int(groupid))

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

	// goes to redis
	sum := md5.Sum(token)
	storageToken := base64.StdEncoding.EncodeToString(sum[:])

	// user hash is like user:100
	user_key := fmt.Sprintf("user:%d", userid)

	// check to see if session exists already
	result, err := cache.HGet(user_key, "session")
	if err != nil && err != ErrCacheMiss {
		return
	}

	old_session_key := fmt.Sprintf("session:%s", result)

	// delete keys
	err = cache.Delete(user_key, old_session_key)
	if err != nil {
		return
	}

	// set in user hash
	err = cache.HMSet(user_key, "session", []byte(storageToken))
	if err != nil {
		return
	}

	// set user group
	err = cache.HMSet(user_key, "group", []byte(gid))
	if err != nil {
		return
	}

	new_session_key := fmt.Sprintf("session:%s", storageToken)

	// set key in redis
	err = cache.SetEx(new_session_key, 31556952, []byte(uid))
	if err != nil {
		return
	}

	// goes in the cookie
	cookieToken = base64.URLEncoding.EncodeToString(token)

	return

}

// validate compares provided session id to redis
func ValidateSession(key []byte) (uid, gid uint, err error) {

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
	cookie_uid := string(token[:index])

	// hash token
	sum := md5.Sum(token)

	// base64 encode sum
	providedHash := base64.StdEncoding.EncodeToString(sum[:])

	// session key in redis
	session_key := fmt.Sprintf("session:%s", providedHash)

	// check for match in user session hash
	userid, err := cache.Get(session_key)
	if err != nil {
		return
	}

	// check if uid matches
	if cookie_uid != string(userid) {
		return e.ErrInvalidSession
	}

	// user hash is like user:100
	user_key := fmt.Sprintf("user:%s", cookie_uid)

	// get group id from user session hash
	groupid, err := cache.HGet(user_key, "group")
	if err != nil {
		return
	}

	// parse user to uint
	uid, err = strconv.ParseUint(cookie_uid, 10, 0)
	if err != nil {
		return
	}

	// parse group to uint
	gid, err = strconv.ParseUint(groupid, 10, 0)
	if err != nil {
		return
	}

	return

}
