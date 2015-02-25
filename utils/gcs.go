package utils

import (
	"bytes"
	"errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
	storage "google.golang.org/api/storage/v1"
	"io/ioutil"

	"github.com/techjanitor/pram-post/config"
)

var gckey []byte

func init() {
	var err error

	gckey, err = ioutil.ReadFile(config.Settings.Google.Key)
	if err != nil {
		panic(err)
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
		return errors.New("problem saving file to gcs")
	}

	return

}

func (i *ImageType) UploadGCS() (err error) {
	buffer := bytes.NewReader(i.image)

	object := &storage.Object{Name: i.Filename}

	service, err := getGCS()
	if err != nil {
		return
	}

	_, err = service.Objects.Insert(config.Settings.Google.Bucket, object).Media(buffer).Do()
	if err != nil {
		return errors.New("problem saving file to gcs")
	}

	return

}

func DeletGCS(object string) (err error) {

	service, err := getGCS()
	if err != nil {
		return
	}

	err := service.Objects.Delete(config.Settings.Google.Bucket, object).Do()
	if err != nil {
		return errors.New("problem deleting gcs file")
	}

	return

}
