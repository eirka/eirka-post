package utils

import (
	"net/url"
)

type Redirect struct {
	Id      uint
	Referer string
	Url     string
	Link    string
}

// Redirect to the correct imageboard after post
func (r *Redirect) Link() (err error) {

	// Get Database handle
	db, err := u.GetDb()
	if err != nil {
		return
	}

	// Get the url of the imageboard from the database
	err = db.QueryRow("SELECT ib_domain FROM imageboards WHERE ib_id = ?", r.Id).Scan(&r.Url)
	if err != nil {
		return
	}

	// Get the scheme from the referer
	parsed, err := url.Parse(r.Referer)
	if err != nil {
		return
	}

	// Create url
	redir := &url.URL{
		Scheme: parsed.Scheme,
		Host:   r.Url,
	}

	// set the link
	r.Link = redir.String()

	return

}
