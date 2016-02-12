package utils

import (
	"github.com/eirka/eirka-libs/config"

	"github.com/eirka/eirka-post/akismet"
)

type Akismet struct {
	Ip      string
	Name    string
	Ua      string
	Referer string
	Comment string
}

// Check comment for spam with akismet
func (c *Akismet) Check() (err error) {

	config := &akismet.Config{
		APIKey:    config.Settings.Akismet.Key,
		Host:      config.Settings.Akismet.Host,
		UserAgent: akismet.UserAgentString("Pram/1.2"),
	}

	// verify the akismet api key
	// TODO: add errors here to a system log
	err = akismet.VerifyKey(config)
	if err != nil {
		return
	}

	comment := akismet.Comment{
		UserIP:    c.Ip,
		UserAgent: c.Ua,
		Content:   c.Comment,
		Referrer:  c.Referer,
		Type:      "comment",
	}

	// check the comment with akismet
	err = akismet.CommentCheck(config, comment)
	if err != nil {
		return
	}

	return

}
