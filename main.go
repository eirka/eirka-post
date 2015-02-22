package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"runtime"

	"github.com/techjanitor/pram-post/config"
	c "github.com/techjanitor/pram-post/controllers"
	m "github.com/techjanitor/pram-post/middleware"
	u "github.com/techjanitor/pram-post/utils"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	config.Print()

	// Set up DB connection
	u.NewDb()

	// Set up Redis connection
	u.NewRedisCache()
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

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Settings.General.Address, config.Settings.General.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}
