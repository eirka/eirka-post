package akismet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// UserAgentString constructs a user agent string suitable for use with akismet,
// based on their recommendations here.
// See 'Setting your user agent' for more information:
// http://akismet.com/development/api/#getting-started
func UserAgentString(application string) string {
	return fmt.Sprintf("%v | Akismet-Go/1.0.0", application)
}

// Config is a struct containing akismet configuration unique to each application.
type Config struct {
	APIKey    string
	Host      string
	UserAgent string
}

// VerifyKeyURL returns the akismet api's key verification URL
func (c *Config) VerifyKeyURL() string {
	return "http://rest.akismet.com/1.1/verify-key"
}

// CommentCheckURL returns the akismet api's comment check URL
func (c *Config) CommentCheckURL() string {
	return fmt.Sprintf("http://%v.rest.akismet.com/1.1/comment-check", c.APIKey)
}

// SubmitSpamURL returns the akismet api's spam submission URL
func (c *Config) SubmitSpamURL() string {
	return fmt.Sprintf("http://%v.rest.akismet.com/1.1/submit-spam", c.APIKey)
}

// SubmitSpamURL returns the akismet api's ham submission URL
func (c *Config) SubmitHamURL() string {
	return fmt.Sprintf("http://%v.rest.akismet.com/1.1/submit-ham", c.APIKey)
}

// Comment represents a single user comment to be checked and submitted with
// Akismet. The UserIP and UserAgent fields are the only two required fields,
// although for best results you should fill in as much of this info as possible.
// The fields are explained in more detail here:
// http://akismet.com/development/api/#comment-check
type Comment struct {
	UserIP      string
	UserAgent   string
	Referrer    string
	Permalink   string
	Type        string
	Author      string
	AuthorEmail string
	AuthorURL   string
	Content     string
}

// MakePOST builds a POST request to a given URL containing the Comment object
// as POST data. It returns the http response, and any errors.
func (comment *Comment) MakePOST(config *Config, url string) (resp *http.Response, err error) {
	client := &http.Client{}
	requestBody := request{
		"key":                  config.APIKey,
		"blog":                 config.Host,
		"user_ip":              comment.UserIP,
		"user_agent":           comment.UserAgent,
		"referrer":             comment.Referrer,
		"permalink":            comment.Permalink,
		"comment_type":         comment.Type,
		"comment_author":       comment.Author,
		"comment_author_email": comment.AuthorEmail,
		"comment_author_url":   comment.AuthorURL,
		"comment_content":      comment.Content,
	}

	req, err := http.NewRequest("POST", url, requestBody.Reader())
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", config.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(req)
	return
}

var (
	ErrSpam           error = errors.New("Comment is spam")
	ErrInvalidRequest error = errors.New("Malformed request")
)

// CommentCheck submits the given comment to Akismet, and
// returns nil if the comment isn't spam; ErrSpam if it is;
// ErrInvalidRequest if the comment or configuration is incorrect;
// and ErrUnknown otherwise.
func CommentCheck(config *Config, comment Comment) error {
	resp, err := comment.MakePOST(config, config.CommentCheckURL())
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	switch bodyStr {
	case "false":
		return nil
	case "true":
		return ErrSpam
	case "invalid":
		return ErrInvalidRequest
	}
	return ErrUnknown
}

// CommentSubmitHam submits the given comment to Akismet as Ham
// Returns nil if the submission was successful, or ErrUnknown
// otherwise.
func CommentSubmitHam(config *Config, comment Comment) error {
	resp, err := comment.MakePOST(config, config.SubmitHamURL())
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	switch bodyStr {
	case "Thanks for making the web a better place.":
		return nil
	}
	return ErrUnknown
}

// CommentSubmitSpam submits the given comment to Akismet as Spam
// Returns nil if the submission was successful, or ErrUnknown
// otherwise.
func CommentSubmitSpam(config *Config, comment Comment) error {
	resp, err := comment.MakePOST(config, config.SubmitHamURL())
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	switch bodyStr {
	case "Thanks for making the web a better place.":
		return nil
	}
	return ErrUnknown
}

// request is a map of key:value pairs to be POSTed to the api
type request map[string]string

// String converts the map to POST friendly "a=b&x=y" strings
func (req request) String() string {
	var requestString string
	var pairs []string
	for k, v := range req {
		pair := fmt.Sprintf("%v=%v", k, v)
		pairs = append(pairs, pair)
	}
	requestString = strings.Join(pairs, "&")
	return requestString
}

// Reader returns a strings.Reader around the value of String()
func (req request) Reader() *strings.Reader {
	return strings.NewReader(req.String())
}

var (
	ErrInvalidKey error = errors.New("Key invalid")
	ErrUnknown    error = errors.New("Unknown response")
)

// VerifyKey checks the configuration with Akismet. This should be performed
// at application startup, otherwise future api calls may fail due to invalid
// configuration
// Returns nil if the configuration is valid;
// returns ErrInvalidKey if it is not;
// returns ErrUnknown otherwise.
func VerifyKey(config *Config) error {
	client := &http.Client{}
	requestBody := request{
		"key":  config.APIKey,
		"blog": config.Host,
	}

	req, err := http.NewRequest("POST", config.VerifyKeyURL(), requestBody.Reader())
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", config.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	switch bodyStr {
	case "valid":
		return nil
	case "invalid":
		return ErrInvalidKey
	}
	return ErrUnknown
}
