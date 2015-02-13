package main

import (
	"github.com/gin-gonic/gin"
	"runtime"

	"pram-post/config"
	c "pram-post/controllers"
	m "pram-post/middleware"
	u "pram-post/utils"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	conf := config.Settings

	db := conf.Database

	// Set up DB connection
	u.NewDb(db.DbUser, db.DbPassword, db.DbProto, db.DbHost, db.DbDatabase, db.DbMaxIdle, db.DbMaxConnections)

	redis := conf.Redis

	// Set up Redis connection
	u.NewRedisCache(redis.RedisAddress, redis.RedisProtocol, redis.RedisMaxIdle, redis.RedisMaxConnections)
}

func main() {
	r := gin.Default()

	r.Use(gin.ForwardedFor("127.0.0.1/32"))
	// Checks for antispam cookie
	r.Use(m.GetAntiSpamCookie())

	r.POST("/thread/new", c.ThreadController)
	r.POST("/thread/reply", c.ReplyController)
	r.POST("/tag/new", c.NewTagController)
	r.POST("/tag/add", c.AddTagController)
	r.NoRoute(c.ErrorController)

	r.Run("127.0.0.1:5015")

}
