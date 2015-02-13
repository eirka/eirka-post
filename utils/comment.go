package utils

import (
	"github.com/techjanitor/pram-post/akismet"
	"github.com/techjanitor/pram-post/config"
)

type CheckComment struct {
	Ip      string
	Name    string
	Ua      string
	Referer string
	Comment string
}

// Check comment for spam with akismet
func (c *CheckComment) Get() (err error) {

	err = CheckStopForumSpam(c.Ip)
	if err != nil {
		return
	}

	config := &akismet.Config{
		APIKey:    config.Settings.Akismet.AkismetKey,
		Host:      config.Settings.Akismet.AkismetHost,
		UserAgent: akismet.UserAgentString("Pram/1.2"),
	}

	if c.Name == "Anonymous" {
		c.Name = ""
	}

	err = akismet.VerifyKey(config)
	if err != nil {
		return
	}

	comment := akismet.Comment{
		UserIP:    c.Ip,
		UserAgent: c.Ua,
		Content:   c.Comment,
		Author:    c.Name,
		Referrer:  c.Referer,
		Type:      "comment",
	}

	err = akismet.CommentCheck(config, comment)
	if err != nil {
		return
	}

	return

}
