package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"

	"github.com/techjanitor/pram-libs/auth"
	"github.com/techjanitor/pram-libs/config"
	"github.com/techjanitor/pram-libs/cors"
	"github.com/techjanitor/pram-libs/db"
	"github.com/techjanitor/pram-libs/redis"

	local "github.com/techjanitor/pram-post/config"
	c "github.com/techjanitor/pram-post/controllers"
	m "github.com/techjanitor/pram-post/middleware"
	u "github.com/techjanitor/pram-post/utils"
)

var (
	version = "0.9.1"
)

func init() {

	// Database connection settings
	dbase := db.Database{

		User:           local.Settings.Database.User,
		Password:       local.Settings.Database.Password,
		Proto:          local.Settings.Database.Proto,
		Host:           local.Settings.Database.Host,
		Database:       local.Settings.Database.Database,
		MaxIdle:        local.Settings.Database.MaxIdle,
		MaxConnections: local.Settings.Database.MaxConnections,
	}

	// Set up DB connection
	dbase.NewDb()

	// Get limits and stuff from database
	config.GetDatabaseSettings()

	// redis settings
	r := redis.Redis{
		// Redis address and max pool connections
		Protocol:       local.Settings.Redis.Protocol,
		Address:        local.Settings.Redis.Address,
		MaxIdle:        local.Settings.Redis.MaxIdle,
		MaxConnections: local.Settings.Redis.MaxConnections,
	}

	// Set up Redis connection
	r.NewRedisCache()

	// set auth middleware secret
	auth.Secret = local.Settings.Session.Secret

	// print the starting info
	StartInfo()

	// Print out config
	config.Print()

	// Print out config
	local.Print()

	// check what services are available
	u.CheckServices()

	// Print capabilities
	u.Services.Print()

	// set cors domains
	cors.SetDomains(local.Settings.CORS.Sites, strings.Split("POST", ","))

}

func main() {
	r := gin.Default()

	r.Use(cors.CORS())

	r.NoRoute(c.ErrorController)

	// all users
	public := r.Group("/")
	public.Use(auth.Auth(false))

	public.POST("/thread/new", m.GetAntiSpamCookie(), c.ThreadController)
	public.POST("/thread/reply", m.GetAntiSpamCookie(), c.ReplyController)
	public.POST("/tag/new", m.GetAntiSpamCookie(), c.NewTagController)
	public.POST("/tag/add", m.GetAntiSpamCookie(), c.AddTagController)
	public.POST("/register", c.RegisterController)
	public.POST("/login", c.LoginController)

	// requires user perms
	users := r.Group("/user")
	users.Use(auth.Auth(true))

	users.POST("/favorite", c.FavoritesController)
	users.POST("/password", c.PasswordController)
	users.POST("/email", c.EmailController)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", local.Settings.Post.Address, local.Settings.Post.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}

func StartInfo() {

	fmt.Println(strings.Repeat("*", 60))
	fmt.Printf("%-20v\n\n", "PRAM-POST")
	fmt.Printf("%-20v%40v\n", "Version", version)
	fmt.Println(strings.Repeat("*", 60))

}
