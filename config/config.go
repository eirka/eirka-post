package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var Settings *Config

type Config struct {
	General struct {
		// Settings for daemon
		Address string
		Port    uint

		// Default name if none specified
		DefaultName string

		// Storage directory for images
		ImageDir     string
		ThumbnailDir string
	}

	// sites for CORS
	CORS struct {
		Sites []string
	}

	Database struct {
		// Database connection settings
		User           string
		Password       string
		Proto          string
		Host           string
		Database       string
		MaxIdle        int
		MaxConnections int
	}
	Redis struct {
		// Redis address and max pool connections
		Protocol       string
		Address        string
		MaxIdle        int
		MaxConnections int
	}

	// settings for google storage
	Google struct {
		Auth   string
		Bucket string
		Key    string
	}

	// settings for amazon s3
	Amazon struct {
		Region   string
		Endpoint string
		Bucket   string
		Id       string
		Key      string
	}

	Akismet struct {
		// Akismet settings
		Key  string
		Host string
	}

	Antispam struct {
		// Antispam Key from Prim
		AntispamKey string

		// Antispam cookie
		CookieName  string
		CookieValue string
	}

	// HMAC secret for bcrypt
	Session struct {
		Secret string
	}

	StopForumSpam struct {
		// Stop Forum Spam settings
		Confidence float64
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

func Print() {

	// Marshal the structs into JSON
	output, err := json.MarshalIndent(Settings, "", "  ")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", output)

}

func init() {
	file, err := os.Open("/etc/pram/post.conf")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Settings = &Config{}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&Settings)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
