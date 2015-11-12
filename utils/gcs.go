package utils

import (
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	storage "google.golang.org/api/storage/v1"
	"io/ioutil"
	"os"

	"github.com/techjanitor/pram-libs/config"
)

var gckey []byte

func init() {
	var err error

	if Services.Storage.Google {
		gckey, err = ioutil.ReadFile(config.Settings.Google.Key)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

}

// Authenticate to Google Cloud Storage and return handler
func getGCS() (service *storage.Service, err error) {

	authconf := &jwt.Config{
		Email:      config.Settings.Google.Auth,
		PrivateKey: gckey,
		Scopes:     []string{storage.DevstorageReadWriteScope},
		TokenURL:   "https://accounts.google.com/o/oauth2/token",
	}

	client := authconf.Client(oauth2.NoContext)

	service, err = storage.New(client)
	if err != nil {
		return nil, errors.New("problem saving file to gcs")
	}

	return

}

// Upload a file to Google Cloud Storage
func UploadGCS(filepath, filename string) (err error) {

	service, err := getGCS()
	if err != nil {
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		return errors.New("problem opening file for gcs")
	}
	defer file.Close()

	object := &storage.Object{
		Name:         filename,
		CacheControl: "public, max-age=31536000",
	}

	_, err = service.Objects.Insert(config.Settings.Google.Bucket, object).Media(file).Do()
	if err != nil {
		return
	}

	return

}

// Delete a file from Google Cloud Storage
func DeleteGCS(object string) (err error) {

	service, err := getGCS()
	if err != nil {
		return
	}

	err = service.Objects.Delete(config.Settings.Google.Bucket, object).Do()
	if err != nil {
		return errors.New("problem deleting gcs file")
	}

	return

}
