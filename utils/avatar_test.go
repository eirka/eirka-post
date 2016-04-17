package utils

import (
	"os"
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	"github.com/stretchr/testify/assert"

	local "github.com/eirka/eirka-post/config"
)

func TestSaveAvatar(t *testing.T) {

	// Database connection settings
	dbase := db.Database{

		User:           local.Settings.Database.User,
		Password:       local.Settings.Database.Password,
		Proto:          local.Settings.Database.Proto,
		Host:           local.Settings.Database.Host,
		Database:       local.Settings.Database.Database,
		MaxIdle:        local.Settings.Database.MaxIdle,
		MaxConnections: local.Settings.Database.MaxConnections,
	}

	// Set up DB connection
	dbase.NewDb()

	// Get limits and stuff from database
	config.GetDatabaseSettings()

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	img.Ib = 1

	err := img.SaveAvatar()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img.SHA, "SHA should be returned")
		assert.Equal(t, img.Ext, ".jpg", "Ext should be the same")
		assert.Equal(t, img.mime, "image/jpeg", "Mime type should be the same")
		assert.Equal(t, img.OrigHeight, 300, "Height should be the same")
		assert.Equal(t, img.OrigWidth, 300, "Width should be the same")
		assert.Equal(t, img.ThumbHeight, 128, "Thumbnail height should be returned")
		assert.Equal(t, img.ThumbWidth, 128, "Thumbnail width should be returned")
		assert.NotEmpty(t, img.Filename, "Filename should be returned")
		assert.NotEmpty(t, img.Thumbnail, "Thumbnail name should be returned")
	}

	file, err := os.Open(img.Filepath)
	assert.NoError(t, err, "An error was not expected")

	fileinfo, err := file.Stat()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, fileinfo.Name(), img.Filename, "Name should be the same")
	}

	thumb, err := os.Open(img.Thumbpath)
	assert.NoError(t, err, "An error was not expected")

	thumbinfo, err := thumb.Stat()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, thumbinfo.Name(), img.Thumbnail, "Name should be the same")
	}

}

func TestGenerateAvatar(t *testing.T) {

	err := GenerateAvatar(0)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(1)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(2)
	assert.NoError(t, err, "An error was not expected")

}
