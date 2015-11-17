package utils

import (
	"net/url"

	"github.com/techjanitor/pram-libs/db"
)

// Redirect to the correct imageboard after post
func Link(id uint) (err error) (host string, err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	// Get the url of the imageboard from the database
	err = dbase.QueryRow("SELECT ib_domain FROM imageboards WHERE ib_id = ?", r.Id).Scan(&host)
	if err != nil {
		return
	}

	return

}
