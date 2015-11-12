package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/techjanitor/pram-libs/config"
)

// Stop Forum Spam return format
type StopForumSpam struct {
	Ip struct {
		Appears    float64 `json:"appears"`
		Confidence float64 `json:"confidence"`
		Frequency  float64 `json:"frequency"`
		Lastseen   string  `json:"lastseen"`
	} `json:"ip"`
	Success float64 `json:"success"`
}

// Check Stop Forum Spam blacklist for IP
func CheckStopForumSpam(ip string) error {

	queryValues := url.Values{}

	queryValues.Add("ip", ip)
	queryValues.Add("f", "json")

	sfs_endpoint := &url.URL{
		Scheme:   "http",
		Host:     "api.stopforumspam.org",
		Path:     "api",
		RawQuery: queryValues.Encode(),
	}

	res, err := http.Get(sfs_endpoint.String())
	if err != nil {
		return errors.New("error reaching sfs")
	}
	defer res.Body.Close()

	res.Header.Add("User-Agent", "Pram/1.2")

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("error parsing sfs response")
	}

	sfs_data := StopForumSpam{}

	err = json.Unmarshal(body, &sfs_data)
	if err != nil {
		return errors.New("error parsing sfs data")
	}

	if sfs_data.Ip.Confidence > config.Settings.StopForumSpam.Confidence {
		return errors.New("ip on blacklist")
	}

	return nil

}
