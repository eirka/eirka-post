package utils

import (
	"net/url"
)

// Redirect to the referer host if there is one
func RedirectLink(refer string) string {

	parsed, err := url.Parse(refer)
	if err != nil {
		return "/"
	}

	redir := &url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
	}

	return redir.String()

}
