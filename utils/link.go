package utils

import (
	"fmt"
	"net/url"
)

// Redirect to the referer host if there is one
func RedirectLink(refer string) (link string) {

	link = "/"

	parsed, err := url.Parse(refer)
	if err != nil {
		return
	}

	if parsed != nil {
		link = fmt.Sprintf("//%s/", parsed.Host)
	}

	return

}
