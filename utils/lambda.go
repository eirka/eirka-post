package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/techjanitor/pram-libs/config"
)

// the format for the lambda thumbnail
type LambdaThumbnail struct {
	Bucket    string `json:"bucket"`
	Filename  string `json:"filename"`
	Thumbnail string `json:"thumbnail"`
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
}

// the format of the lambda context response
type LambdaResponse struct {
	Success string `json:"successMessage"`
	Error   string `json:"errorMessage"`
}

// posts to our api endpoint
func (t *LambdaThumbnail) Create() (err error) {

	// Marshal the structs into JSON
	output, err := json.Marshal(t)
	if err != nil {
		return
	}

	b := bytes.NewReader(output)

	// Make a post request with our writer
	req, err := http.NewRequest("POST", config.Settings.Lambda.Thumbnail.Endpoint, b)
	if err != nil {
		return
	}

	req.Header.Add("x-api-key", config.Settings.Lambda.Thumbnail.Key)
	req.Header.Add("User-Agent", "Pram/1.2")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	response := LambdaResponse{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	if response.Error != "" {
		return errors.New(fmt.Sprintf("Error creating thumbnail: %s", response.Error))
	}

	return

}
