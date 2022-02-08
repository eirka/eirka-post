package middleware

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/gin-gonic/gin"
)

// Scamalytics check ip with scamalytics
func Scamalytics() gin.HandlerFunc {
	return func(c *gin.Context) {

		// check ip against scamalytics
		err := CheckScamalytics(c.ClientIP())
		if err == e.ErrBlacklist {
			c.JSON(http.StatusBadRequest, gin.H{"error_message": "IP is on spam blacklist"})
			c.Error(err).SetMeta("Scamalytics.CheckScamalytics")
			c.Abort()
			return
		}

		c.Next()

	}
}

type ScamalyticsResponse struct {
	Status  string `json:"status"`
	Mode    string `json:"mode"`
	IP      string `json:"ip"`
	Score   int    `json:"score"`
	Risk    string `json:"risk"`
	URL     string `json:"url"`
	Credits struct {
		Used                        int    `json:"used"`
		Remaining                   int    `json:"remaining"`
		LastSyncTimestampUtc        string `json:"last_sync_timestamp_utc"`
		SecondsElapsedSinceLastSync int    `json:"seconds_elapsed_since_last_sync"`
		Note                        string `json:"note"`
	} `json:"credits"`
	Exec string `json:"exec"`
}

// CheckScamalytics will query blacklist api for IP
func CheckScamalytics(ip string) (err error) {

	if len(ip) == 0 {
		return errors.New("no ip provided")
	}

	queryValues := url.Values{}

	queryValues.Set("ip", ip)
	queryValues.Set("key", config.Settings.Scamalytics.Key)

	// construct the api request
	scamalyticsEndpoint := &url.URL{
		Scheme:   "https",
		Host:     config.Settings.Scamalytics.Endpoint,
		Path:     config.Settings.Scamalytics.Path,
		RawQuery: queryValues.Encode(),
	}

	// our http request
	req, err := http.NewRequest(http.MethodGet, scamalyticsEndpoint.String(), nil)
	if err != nil {
		return errors.New("error creating scamalytics request")
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
		return errors.New("error reaching scamalytics")
	}
	defer resp.Body.Close()

	// read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.New("error parsing scamalytics response")
	}

	scamalyticsData := ScamalyticsResponse{}

	// unmarshal into struct
	err = json.Unmarshal(body, &scamalyticsData)
	if err != nil {
		return errors.New("error parsing scamalytics data")
	}

	// check if the spammer confidence level is over our setting
	if scamalyticsData.Score > config.Settings.Scamalytics.Score {
		return e.ErrBlacklist
	}

	return
}
