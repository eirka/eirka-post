package utils

import (
	"fmt"
	"net/url"
	"path"

	"github.com/eirka/eirka-libs/db"
)

// Link will create the redirect to the correct imageboard after post
func Link(id uint, referer string) (host string, err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// Get the url of the imageboard from the database
	err = dbase.QueryRow("SELECT ib_domain FROM imageboards WHERE ib_id = ?", id).Scan(&host)
	if err != nil {
		return
	}

	// for subdirectory support
	if path.Dir(host) != "." {
		host = fmt.Sprintf("%s/b/%s", path.Dir(host), path.Base(host))
	}

	// Get the scheme from the referer
	refer, err := url.Parse(referer)
	if err != nil {
		return
	}

	// Get the domain from the database
	base, err := url.Parse(host)
	if err != nil {
		return
	}

	// Create url
	redir := &url.URL{
		Scheme: refer.Scheme,
		Path:   base.Path,
	}

	// set the link
	host = fmt.Sprintf("%s/", redir)

	return

}
