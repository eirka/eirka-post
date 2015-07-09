package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"

	"github.com/techjanitor/pram-post/config"
	c "github.com/techjanitor/pram-post/controllers"
	m "github.com/techjanitor/pram-post/middleware"
	u "github.com/techjanitor/pram-post/utils"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Set up DB connection
	u.NewDb()

	// Set up Redis connection
	u.NewRedisCache()

	// Get limits and stuff from database
	u.GetDatabaseSettings()

	// Print out config
	config.Print()

}

func main() {
	r := gin.Default()

	// Adds CORS headers
	r.Use(m.CORS())
	// Checks for antispam cookie
	r.Use(m.GetAntiSpamCookie())
	// use auth system
	r.Use(m.Auth(m.SetAuthLevel().All()))

	r.POST("/thread/new", c.ThreadController)
	r.POST("/thread/reply", c.ReplyController)
	r.POST("/tag/new", c.NewTagController)
	r.POST("/tag/add", c.AddTagController)
	r.POST("/register", c.RegisterController)
	r.NoRoute(c.ErrorController)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Settings.General.Address, config.Settings.General.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}
