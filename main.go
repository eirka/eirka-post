package main

import (
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/gin-gonic/gin"
	"net/http"
	"runtime"

	"github.com/techjanitor/pram-libs/auth"
	"github.com/techjanitor/pram-libs/config"
	"github.com/techjanitor/pram-libs/cors"
	"github.com/techjanitor/pram-libs/db"
	"github.com/techjanitor/pram-libs/validate"

	local "github.com/techjanitor/pram-post/config"
	c "github.com/techjanitor/pram-post/controllers"
	m "github.com/techjanitor/pram-post/middleware"
	u "github.com/techjanitor/pram-post/utils"
)

func init() {

	dbase := db.Database{
		// Database connection settings
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
	db.GetDatabaseSettings()

	// Set up Redis connection
	u.NewRedisCache()

	// Print out config
	config.Print()

	// Print out config
	local.Print()

	// set auth middleware secret
	auth.Secret = local.Settings.Session.Secret

	// set cors domains
	cors.Domains = local.Settings.CORS.Sites

	// Print capabilities
	u.Services.Print()

}

func main() {
	r := gin.Default()

	r.Use(cors.CORS())

	r.NoRoute(c.ErrorController)

	// all users
	public := r.Group("/")
	public.Use(auth.Auth(auth.All))

	public.POST("/thread/new", m.GetAntiSpamCookie(), c.ThreadController)
	public.POST("/thread/reply", m.GetAntiSpamCookie(), c.ReplyController)
	public.POST("/tag/new", m.GetAntiSpamCookie(), c.NewTagController)
	public.POST("/tag/add", m.GetAntiSpamCookie(), c.AddTagController)
	public.POST("/register", c.RegisterController)
	public.POST("/login", c.LoginController)

	// requires user perms
	users := r.Group("/user")
	users.Use(auth.Auth(auth.Registered))

	users.POST("/favorite", c.FavoritesController)
	users.POST("/password", c.PasswordController)
	users.POST("/email", c.EmailController)

	// requires mod perms
	mod := r.Group("/mod")
	mod.Use(validate.ValidateParams())
	mod.Use(auth.Auth(auth.Moderators))

	mod.DELETE("/tag/:id", c.DeleteTagController)
	mod.DELETE("/imagetag/:image/:tag", c.DeleteImageTagController)
	mod.DELETE("/thread/:id", c.DeleteThreadController)
	mod.DELETE("/post/:thread/:id", c.DeletePostController)
	mod.POST("/sticky/:thread", c.StickyThreadController)
	mod.POST("/close/:thread", c.CloseThreadController)

	// requires admin perms
	admin := r.Group("/admin")
	admin.Use(validate.ValidateParams())
	admin.Use(auth.Auth(auth.Admins))

	admin.DELETE("/thread/:id", c.PurgeThreadController)
	admin.DELETE("/post/:thread/:id", c.PurgePostController)
	//admin.POST("/ban/:ip", c.BanIpController)
	//admin.DELETE("/flushcache", c.DeleteCacheController)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", local.Settings.Post.Address, local.Settings.Post.Port),
		Handler: r,
	}

	gracehttp.Serve(s)

}
