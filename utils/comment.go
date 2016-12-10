package utils

import (
	"github.com/eirka/eirka-libs/config"

	"github.com/eirka/eirka-post/akismet"
)

// Akismet holds information for an akismet query
type Akismet struct {
	IP      string
	Name    string
	Ua      string
	Referer string
	Comment string
}

// Check comment for spam with akismet
func (c *Akismet) Check() (err error) {

	// noop if akismet is not configured
	if !config.Settings.Akismet.Configured {
		return
	}

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
		UserIP:    c.IP,
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
