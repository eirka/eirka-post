package config

import (
	"encoding/json"
	"os"
)

var Settings *Config

type Config struct {
	Database struct {
		// Database connection settings
		DbUser           string
		DbPassword       string
		DbProto          string
		DbHost           string
		DbDatabase       string
		DbMaxIdle        int
		DbMaxConnections int
	}
	Redis struct {
		// Redis address and max pool connections
		RedisProtocol       string
		RedisAddress        string
		RedisMaxIdle        int
		RedisMaxConnections int
	}

	Akismet struct {
		// Akismet settings
		AkismetKey  string
		AkismetHost string
	}

	Antispam struct {
		// Antispam Key from Prim
		AntispamKey string

		// Antispam cookie
		CookieName  string
		CookieValue string
	}

	StopForumSpam struct {
		// Stop Forum Spam settings
		SfsConfidence float64
	}

	Directory struct {
		// Storage directory for images
		ImageDir     string
		ThumbnailDir string
	}

	General struct {
		// Default name if none specified
		DefaultName string
	}

	Limits struct {
		// Image settings
		ImageMinWidth  int
		ImageMinHeight int
		ImageMaxWidth  int
		ImageMaxHeight int
		ImageMaxSize   int
		WebmMaxLength  int

		// Max posts in a thread
		PostsMax uint

		// Lengths for posting
		CommentMaxLength int
		CommentMinLength int
		TitleMaxLength   int
		TitleMinLength   int
		NameMaxLength    int
		NameMinLength    int
		TagMaxLength     int
		TagMinLength     int

		// Max thumbnail sizes
		ThumbnailMaxWidth  int
		ThumbnailMaxHeight int

		// Max request parameter input size
		ParamMaxSize uint
	}
}

func init() {
	file, err := os.Open("/etc/pram/post.conf")
	if err != nil {
		panic(err)
	}

	Settings = &Config{}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&Settings)
	if err != nil {
		panic(err)
	}

}
