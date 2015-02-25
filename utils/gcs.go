package utils

import (
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	storage "google.golang.org/api/storage/v1"
	"io/ioutil"
	"os"

	"github.com/techjanitor/pram-post/config"
)

var gckey []byte

func init() {
	var err error

	gckey, err = ioutil.ReadFile(config.Settings.Google.Key)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func getGCS() (service *storage.Service, err error) {

	authconf := &jwt.Config{
		Email:      config.Settings.Google.Auth,
		PrivateKey: gckey,
		Scopes:     []string{storage.DevstorageRead_writeScope},
		TokenURL:   "https://accounts.google.com/o/oauth2/token",
	}

	client := authconf.Client(oauth2.NoContext)

	service, err = storage.New(client)
	if err != nil {
		return nil, errors.New("problem saving file to gcs")
	}

	return

}

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

	_, err = service.Objects.Insert(config.Settings.Google.Bucket, &storage.Object{Name: filename}).Media(file).PredefinedAcl("publicRead").Do()
	if err != nil {
		return
	}

	return

}

func DeletGCS(object string) (err error) {

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
