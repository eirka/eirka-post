package utils

import (
	"github.com/techjanitor/pram-libs/db"
)

// Redirect to the correct imageboard after post
func Link(id uint) (host string, err error) {

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

	host = "//" + derp

	return

}
