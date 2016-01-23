package utils

import (
	"bytes"
	crand "crypto/rand"
	"errors"
	"github.com/stretchr/testify/assert"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"

	local "github.com/eirka/eirka-post/config"
)

func init() {

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
}

func testPng(size int) *bytes.Buffer {

	output := new(bytes.Buffer)

	myimage := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{size, size}})

	// This loop just fills the image with random data
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			c := color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255}
			myimage.Set(x, y, c)
		}
	}

	png.Encode(output, myimage)

	return output
}

func testJpeg(size int) *bytes.Buffer {

	output := new(bytes.Buffer)

	myimage := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{size, size}})

	// This loop just fills the image with random data
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			c := color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255}
			myimage.Set(x, y, c)
		}
	}

	jpeg.Encode(output, myimage, nil)

	return output
}

func testRandom() []byte {
	bytes := make([]byte, 20000)

	if _, err := io.ReadFull(crand.Reader, bytes); err != nil {
		panic(err)
	}

	return bytes
}

func formJpegRequest(size int, filename string) *http.Request {

	var b bytes.Buffer

	w := multipart.NewWriter(&b)

	fw, _ := w.CreateFormFile("file", filename)

	io.Copy(fw, testJpeg(size))

	w.Close()

	req, _ := http.NewRequest("POST", "/reply", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req
}

func formRandomRequest(filename string) *http.Request {

	var b bytes.Buffer

	w := multipart.NewWriter(&b)

	fw, _ := w.CreateFormFile("file", filename)

	io.Copy(fw, bytes.NewReader(testRandom()))

	w.Close()

	req, _ := http.NewRequest("POST", "/reply", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	return req
}

func TestIsAllowedExt(t *testing.T) {

	assert.False(t, isAllowedExt(".png.exe"), "Should not be allowed")

	assert.False(t, isAllowedExt(".exe.png"), "Should not be allowed")

	assert.False(t, isAllowedExt(""), "Should not be allowed")

	assert.False(t, isAllowedExt("."), "Should not be allowed")

	assert.False(t, isAllowedExt(".pdf"), "Should not be allowed")

	assert.True(t, isAllowedExt(".jpg"), "Should be allowed")

	assert.True(t, isAllowedExt(".JPEG"), "Should be allowed")

}

func TestCheckReqGoodExt(t *testing.T) {

	var err error

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err = img.checkReqExt()
	assert.NoError(t, err, "An error was not expected")

}

func TestCheckReqBadExt(t *testing.T) {

	req := formJpegRequest(300, "test.crap")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Format not supported"), "Error should match")
	}

}

func TestCheckReqBadExtExploit1(t *testing.T) {

	req := formRandomRequest("test.exe.png")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	assert.NoError(t, err, "An error was not expected")

	err = img.getMD5()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
	}

	err = img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Unknown file type"), "Error should match")
	}

}

func TestCheckReqBadExtExploit2(t *testing.T) {

	req := formRandomRequest("test.png.exe")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Format not supported"), "Error should match")
	}

}

func TestCheckReqNoExt(t *testing.T) {

	req := formJpegRequest(300, "test")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("No file extension"), "Error should match")
	}

}

func TestGetMD5(t *testing.T) {

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.getMD5()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
	}

}

func TestCheckMagicGood(t *testing.T) {

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.getMD5()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
	}

	err = img.checkMagic()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, img.mime, "image/jpeg", "Mime type should be the same")
	}

}

func TestCheckMagicBad(t *testing.T) {

	req := formRandomRequest("test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.getMD5()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
	}

	err = img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Unknown file type"), "Error should match")
	}

}

func TestGetStatsGoodPng(t *testing.T) {

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 3000000

	img := ImageType{}

	img.image = testPng(400)

	err := img.getStats()
	assert.NoError(t, err, "An error was not expected")

}

func TestGetStatsGoodJpeg(t *testing.T) {

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 3000000

	img := ImageType{}

	img.image = testJpeg(400)

	err := img.getStats()
	assert.NoError(t, err, "An error was not expected")

}

func TestGetStatsBadSize(t *testing.T) {

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 3000

	img := ImageType{}

	img.image = testPng(400)

	err := img.getStats()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Image size too large"), "Error should match")
	}

}

func TestGetStatsBadMin(t *testing.T) {

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 300000

	img := ImageType{}

	img.image = testPng(50)

	err := img.getStats()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Image width too small"), "Error should match")
	}

}

func TestGetStatsBadMax(t *testing.T) {

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 300000

	img := ImageType{}

	img.image = testPng(1200)

	err := img.getStats()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("Image width too large"), "Error should match")
	}

}

func TestMakeFilenames(t *testing.T) {

	img := ImageType{}

	img.makeFilenames()

	assert.NotEmpty(t, img.Filename, "Filename should be returned")

	assert.NotEmpty(t, img.Thumbnail, "Thumbnail name should be returned")

}

func TestSaveFile(t *testing.T) {

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.SaveImage()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("No imageboard set on duplicate check"), "Error should match")
	}

	img.Ib = 1

	err = img.SaveImage()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.Equal(t, img.Ext, ".jpg", "Ext should be the same")
		assert.Equal(t, img.mime, "image/jpeg", "Mime type should be the same")
		assert.Equal(t, img.OrigHeight, 300, "Height should be the same")
		assert.Equal(t, img.OrigWidth, 300, "Width should be the same")
		assert.NotZero(t, img.ThumbHeight, "Thumbnail height should be returned")
		assert.NotZero(t, img.ThumbWidth, "Thumbnail width should be returned")
		assert.NotEmpty(t, img.Filename, "Filename should be returned")
		assert.NotEmpty(t, img.Thumbnail, "Thumbnail name should be returned")
	}

	filesize := img.image.Len()

	file, err := os.Open(filepath.Join(local.Settings.Directories.ImageDir, img.Filename))
	assert.NoError(t, err, "An error was not expected")

	info, err := file.Stat()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, info.Name(), img.Filename, "Name should be the same")
		assert.Equal(t, info.Size(), int64(filesize), "Size should be the same")
	}
}
