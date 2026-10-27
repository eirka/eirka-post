package utils

import (
	"errors"
	"log"

	"github.com/eirka/eirka-libs/config"

	"github.com/eirka/eirka-post/akismet"
)

// ErrAkismetFailed is returned when the akismet check could not be completed.
// Controllers send Check errors to the client verbatim, so the underlying
// library error (which can carry request URLs and network details) is logged
// and never returned.
var ErrAkismetFailed = errors.New("spam check failed")

// Akismet holds information for an akismet query
type Akismet struct {
	IP      string
	Name    string
	Ua      string
	Referer string
	Comment string
}

// Check comment for spam with akismet. It returns nil, akismet.ErrSpam, or
// ErrAkismetFailed.
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
	err = akismet.VerifyKey(config)
	if err != nil {
		return akismetError(err)
	}

	comment := akismet.Comment{
		UserIP:    c.IP,
		UserAgent: c.Ua,
		Content:   c.Comment,
		Referrer:  c.Referer,
		Type:      "comment",
	}

	// check the comment with akismet
	return akismetError(akismet.CommentCheck(config, comment))

}

// akismetError maps an akismet library error to one that is safe to show a
// client, logging the original
func akismetError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, akismet.ErrSpam) {
		return akismet.ErrSpam
	}

	log.Printf("akismet: %v", err)

	return ErrAkismetFailed
}
