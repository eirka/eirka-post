package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/facebookgo/pidfile"
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

// configure performs production runtime setup: the pid file, database and redis
// connection pools, runtime settings loaded from the database, and CORS domains.
// It runs from main (not an init func) so that test binaries — which import this
// package — do not try to write a pid file or open database/redis sockets.
func configure() {

	// create pid file
	pidfile.SetPidfilePath("/run/eirka/eirka-post.pid")

	err := pidfile.Write()
	if err != nil {
		panic("Could not write pid file")
	}

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

	// set cors domains
	cors.SetDomains(local.Settings.CORS.Sites, strings.Split("POST", ","))

}

// setupRouter builds the gin engine with every route and its middleware.
func setupRouter() *gin.Engine {
	r := gin.Default()

	// c.ClientIP() feeds the ban check, the spam lookups, the login counter and
	// every stored post/tag/audit IP, so it must not be client controlled. gin's
	// default trusts every peer and returns the leftmost X-Forwarded-For entry,
	// which Cloudflare passes through from the client. Instead, trust only nginx
	// on loopback, and only its CF-Connecting-IP header, which Cloudflare always
	// overwrites. Any other peer resolves to its own TCP address.
	r.RemoteIPHeaders = []string{"CF-Connecting-IP"}
	err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	if err != nil {
		panic("Could not set trusted proxies")
	}

	// Limit how much of a multipart upload is buffered in memory; the remainder
	// streams to a temp file, and LimitBody caps the total request body size.
	r.MaxMultipartMemory = 2 << 20 // 2 MiB

	r.Use(cors.CORS())
	// cap the request body before anything reads it; csrf.Verify parses the
	// whole form when the token is not sent in a header. Only the image upload
	// routes take an upload-sized multipart body
	r.Use(m.LimitBody("/thread/new", "/thread/reply", "/user/avatar"))
	// verified the csrf token from the request
	r.Use(csrf.Verify())
	// check the ban list for the ip
	r.Use(m.Bans())
	// log user agent for successful POST requests
	r.Use(m.RequestLogger())

	r.GET("/status", status.StatusController)
	r.NoRoute(c.ErrorController)

	// all users
	public := r.Group("/")
	public.Use(user.Auth(false))

	public.POST("/thread/new", m.Goodnight(), m.StopSpam(), m.Scamalytics(), m.SpamFilter(), c.ThreadController)
	public.POST("/thread/reply", m.Goodnight(), m.StopSpam(), m.Scamalytics(), m.SpamFilter(), c.ReplyController)
	public.POST("/register", m.StopSpam(), m.Scamalytics(), c.RegisterController)
	public.POST("/login", c.LoginController)
	public.POST("/logout", c.LogoutController)

	// new tags group to enforce login
	tags := r.Group("/tag")
	tags.Use(user.Auth(true))
	tags.POST("/new", c.NewTagController)
	tags.POST("/add", c.AddTagController)

	// requires user perms
	users := r.Group("/user")
	users.Use(user.Auth(true))

	users.POST("/avatar", c.AvatarController)
	users.POST("/favorite", c.FavoritesController)
	users.POST("/password", c.PasswordController)
	users.POST("/email", c.EmailController)

	return r
}

func main() {
	configure()

	r := setupRouter()

	s := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", local.Settings.Post.Host, local.Settings.Post.Port),
		ReadHeaderTimeout: 2 * time.Second,
		// Bound a slow request body (slowloris-on-body); combined with LimitBody
		// this caps both the size and the duration of an upload. WriteTimeout is
		// generous because webm processing chains several ffmpeg/convert steps
		// (each bounded separately by their own context timeouts).
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}

	err := gracehttp.Serve(s)
	if err != nil {
		panic("Could not start server")
	}
}
