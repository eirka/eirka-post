package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/cors"
	"github.com/eirka/eirka-libs/csrf"
	"github.com/eirka/eirka-libs/db"
	"github.com/eirka/eirka-libs/redis"
	"github.com/eirka/eirka-libs/status"
	"github.com/eirka/eirka-libs/user"

	local "github.com/eirka/eirka-post/config"
	c "github.com/eirka/eirka-post/controllers"
	m "github.com/eirka/eirka-post/middleware"
)

func init() {

	// Database connection settings
	dbase := db.Database{

		User:           local.Settings.Database.User,
		Password:       local.Settings.Database.Password,
		Proto:          local.Settings.Database.Protocol,
		Host:           local.Settings.Database.Host,
		Database:       local.Settings.Database.Database,
		MaxIdle:        local.Settings.Post.DatabaseMaxIdle,
		MaxConnections: local.Settings.Post.DatabaseMaxConnections,
	}

	// Set up DB connection
	dbase.NewDb()

	// Get limits and stuff from database
	config.GetDatabaseSettings()

	// redis settings
	r := redis.Redis{
		// Redis address and max pool connections
		Protocol:       local.Settings.Redis.Protocol,
		Address:        local.Settings.Redis.Host,
		MaxIdle:        local.Settings.Post.RedisMaxIdle,
		MaxConnections: local.Settings.Post.RedisMaxConnections,
	}

	// Set up Redis connection
	r.NewRedisCache()

	// set auth middleware secret
	user.Secret = local.Settings.Session.Secret

	// set cors domains
	cors.SetDomains(local.Settings.CORS.Sites, strings.Split("POST", ","))

}

func main() {
	r := gin.Default()

	r.Use(cors.CORS())
	// verified the csrf token from the request
	r.Use(csrf.Verify())
	// check the ban list for the ip
	r.Use(m.Bans())

	r.GET("/status", status.StatusController)
	r.NoRoute(c.ErrorController)

	// all users
	public := r.Group("/")
	public.Use(user.Auth(false))

	public.POST("/thread/new", m.StopSpam(), c.ThreadController)
	public.POST("/thread/reply", m.StopSpam(), c.ReplyController)
	public.POST("/tag/new", c.NewTagController)
	public.POST("/tag/add", c.AddTagController)
	public.POST("/register", m.StopSpam(), c.RegisterController)
	public.POST("/login", c.LoginController)

	// requires user perms
	users := r.Group("/user")
	users.Use(user.Auth(true))

	users.POST("/avatar", c.AvatarController)
	users.POST("/favorite", c.FavoritesController)
	users.POST("/password", c.PasswordController)
	users.POST("/email", c.EmailController)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", local.Settings.Post.Host, local.Settings.Post.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}
