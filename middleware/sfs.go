package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
)

// StopSpam check ip with stop forum spam
func StopSpam() gin.HandlerFunc {
	return func(c *gin.Context) {

		// check ip against stop forum spam
		err := CheckStopForumSpam(c.ClientIP())
		if err == e.ErrBlacklist {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": "IP is on spam blacklist"})
			c.Error(err).SetMeta("StopSpam.CheckStopForumSpam")
			c.Abort()
			return
		}

		c.Next()

	}
}

// StopForumSpam is the request return format
type StopForumSpam struct {
	IP struct {
		Appears    float64 `json:"appears"`
		Confidence float64 `json:"confidence"`
		Frequency  float64 `json:"frequency"`
		Lastseen   string  `json:"lastseen"`
	} `json:"ip"`
	Success float64 `json:"success"`
}

// CheckStopForumSpam will query blacklist api for IP
func CheckStopForumSpam(ip string) (err error) {

	if len(ip) == 0 {
		return errors.New("no ip provided")
	}

	queryValues := url.Values{}

	queryValues.Set("ip", ip)
	queryValues.Set("f", "json")

	// construct the api request
	sfsEndpoint := &url.URL{
		Scheme:   "http",
		Host:     "api.stopforumspam.org",
		Path:     "api",
		RawQuery: queryValues.Encode(),
	}

	// our http request
	req, err := http.NewRequest(http.MethodGet, sfsEndpoint.String(), nil)
	if err != nil {
		return errors.New("error creating SFS request")
	}

	// set ua header
	req.Header.Set("User-Agent", "Eirka/1.2")

	// a client with a timeout
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	// do the request
	// TODO: add errors here to a system log
	resp, err := netClient.Do(req)
	if err != nil {
		return errors.New("error reaching SFS")
	}
	defer resp.Body.Close()

	// read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.New("error parsing SFS response")
	}

	sfsData := StopForumSpam{}

	// unmarshal into struct
	err = json.Unmarshal(body, &sfsData)
	if err != nil {
		return errors.New("error parsing SFS data")
	}

	// check if the spammer confidence level is over our setting
	if sfsData.IP.Confidence > config.Settings.StopForumSpam.Confidence {
		return e.ErrBlacklist
	}

	return

}
