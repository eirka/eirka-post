package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

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

	// channel for shutdown
	c := make(chan os.Signal, 10)

	// watch for shutdown signals to shutdown cleanly
	signal.Notify(c, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-c
		Shutdown()
	}()

}

func main() {
	r := gin.Default()

	r.Use(m.CORS())

	r.NoRoute(c.ErrorController)

	// all users
	public := r.Group("/")
	public.Use(m.Auth(m.SetAuthLevel().All()))

	public.POST("/thread/new", m.GetAntiSpamCookie(), c.ThreadController)
	public.POST("/thread/reply", m.GetAntiSpamCookie(), c.ReplyController)
	public.POST("/tag/new", m.GetAntiSpamCookie(), c.NewTagController)
	public.POST("/tag/add", m.GetAntiSpamCookie(), c.AddTagController)
	public.POST("/register", c.RegisterController)
	public.POST("/login", c.LoginController)

	// requires user perms
	users := r.Group("/user")
	users.Use(m.Auth(m.SetAuthLevel().Registered()))

	users.POST("/favorite", c.FavoritesController)
	users.POST("/password", c.PasswordController)
	users.POST("/email", c.EmailController)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Settings.General.Address, config.Settings.General.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}

// called on sigterm or interrupt
func Shutdown() {

	log.Println("Shutting down...")

	// close the database connection
	log.Println("Closing database connection")
	err := u.CloseDb()
	if err != nil {
		log.Println(err)
	}

}
