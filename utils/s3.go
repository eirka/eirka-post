package utils

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"os"

	"github.com/techjanitor/pram-libs/config"
)

// Authenticate to AWS and return handler
func getS3() (svc *session.Session, err error) {

	// new credentials from settings
	creds := credentials.NewStaticCredentials(config.Settings.Amazon.Id, config.Settings.Amazon.Key, "")

	// create our session
	svc = session.New(&aws.Config{
		Region:      aws.String(config.Settings.Amazon.Region),
		Credentials: creds,
	})

	return

}

// Upload a file to S3
func UploadS3(filepath, filename, mime string) (err error) {

	session, err := getS3()
	if err != nil {
		return
	}

	file, err := os.Open(filepath)
	if err != nil {
		return errors.New("problem opening file for s3")
	}
	defer file.Close()

	uploader := s3manager.NewUploader(session)

	params := &s3manager.UploadInput{
		Bucket:               aws.String(config.Settings.Amazon.Bucket),
		Key:                  aws.String(filename),
		Body:                 file,
		ContentType:          aws.String(mime),
		ServerSideEncryption: aws.String(s3.ServerSideEncryptionAes256),
	}

	_, err = uploader.Upload(params)

	return

}

// Delete a file from S3
func DeleteS3(object string) (err error) {

	session, err := getS3()
	if err != nil {
		return
	}

	svc := s3.New(session)

	params := &s3.DeleteObjectInput{
		Bucket: aws.String(config.Settings.Amazon.Bucket),
		Key:    aws.String(object),
	}

	_, err = svc.DeleteObject(params)

	return

}
