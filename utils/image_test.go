package utils

import (
	"bytes"
	crand "crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
)

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
		assert.Equal(t, errors.New("format not supported"), err, "Error should match")
	}

}

func TestCheckReqBadExtExploit1(t *testing.T) {

	req := formRandomRequest("test.exe.png")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	assert.NoError(t, err, "An error was not expected")

	err = img.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.image, "File bytes should be returned")
	}

	err = img.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img.SHA, "SHA should be returned")
	}

	err = img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("unknown file type"), err, "Error should match")
	}

}

func TestCheckReqBadExtExploit2(t *testing.T) {

	req := formRandomRequest("test.png.exe")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("format not supported"), err, "Error should match")
	}

}

func TestCheckReqNoExt(t *testing.T) {

	req := formJpegRequest(300, "test")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.checkReqExt()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("no file extension"), err, "Error should match")
	}

}

func TestCopyBytes(t *testing.T) {

	imagefile := testJpeg(500)

	filesize := imagefile.Len()

	var b bytes.Buffer

	w := multipart.NewWriter(&b)

	fw, _ := w.CreateFormFile("file", "image1.jpg")

	io.Copy(fw, imagefile)

	w.Close()

	req, _ := http.NewRequest("POST", "/reply", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.image, "File bytes should be returned")
	}

	assert.Equal(t, img.image.Len(), filesize, "File sizes should match")

}

func TestGetMD5(t *testing.T) {

	req := formJpegRequest(300, "test.jpeg")

	img1 := ImageType{}

	img1.File, img1.Header, _ = req.FormFile("file")

	err := img1.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img1.image, "File bytes should be returned")
	}

	err = img1.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img1.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img1.SHA, "SHA should be returned")
	}

	req2 := formJpegRequest(300, "test.jpeg")

	img2 := ImageType{}

	img2.File, img2.Header, _ = req2.FormFile("file")

	err = img2.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img2.image, "File bytes should be returned")
	}

	err = img2.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img2.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img2.SHA, "SHA should be returned")
	}

	assert.NotEqual(t, img1.MD5, img2.MD5, "MD5 should not be the same")
	assert.NotEqual(t, img1.SHA, img2.SHA, "SHA should not be the same")

}

// we will check to see if two requests if the same file will generate the same hash
func TestGetMD5Duplicate(t *testing.T) {

	imagefile := testJpeg(500)

	var b bytes.Buffer

	w1 := multipart.NewWriter(&b)

	fw1, _ := w1.CreateFormFile("file", "image1.jpg")

	io.Copy(fw1, bytes.NewReader(imagefile.Bytes()))

	w1.Close()

	req1, _ := http.NewRequest("POST", "/reply", &b)
	req1.Header.Set("Content-Type", w1.FormDataContentType())

	img1 := ImageType{}

	img1.File, img1.Header, _ = req1.FormFile("file")

	err := img1.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img1.image, "File bytes should be returned")
	}

	err = img1.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img1.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img1.SHA, "SHA should be returned")
	}

	b.Reset()

	w2 := multipart.NewWriter(&b)

	fw2, _ := w2.CreateFormFile("file", "image2.jpg")

	io.Copy(fw2, bytes.NewReader(imagefile.Bytes()))

	w2.Close()

	req2, _ := http.NewRequest("POST", "/reply", &b)
	req2.Header.Set("Content-Type", w2.FormDataContentType())

	img2 := ImageType{}

	img2.File, img2.Header, _ = req2.FormFile("file")

	err = img2.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img2.image, "File bytes should be returned")
	}

	err = img2.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img2.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img2.SHA, "SHA should be returned")
	}

	assert.Equal(t, img1.MD5, img2.MD5, "MD5 should be the same")
	assert.Equal(t, img1.SHA, img2.SHA, "SHA should be the same")
	assert.Equal(t, img1.image.Len(), img2.image.Len(), "Size should be the same")
}

func TestCheckBanned(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	nomatch := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_files WHERE ban_hash`).WillReturnRows(nomatch)

	img := ImageType{
		MD5: "banned",
	}

	err = img.checkBanned()
	assert.NoError(t, err, "An error was not expected")

	match := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_files WHERE ban_hash`).WillReturnRows(match)

	err = img.checkBanned()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("file is banned"), err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestCheckDuplicate(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	nomatch := sqlmock.NewRows([]string{"count", "post", "thread"}).AddRow(0, 0, 0)
	mock.ExpectQuery(`select count\(1\),posts.post_num,threads.thread_id from threads`).WillReturnRows(nomatch)

	img := ImageType{
		Ib:  1,
		MD5: "test",
	}

	err = img.checkDuplicate()
	assert.NoError(t, err, "An error was not expected")

	match := sqlmock.NewRows([]string{"count", "post", "thread"}).AddRow(1, 10, 2)
	mock.ExpectQuery(`select count\(1\),posts.post_num,threads.thread_id from threads`).WillReturnRows(match)

	err = img.checkDuplicate()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("image has already been posted. Thread: 2 Post: 10"), err, "Error should match")
	}

	assert.NoError(t, mock.ExpectationsWereMet(), "An error was not expected")

}

func TestCheckMagicGood(t *testing.T) {

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.image, "File bytes should be returned")
	}

	err = img.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img.SHA, "SHA should be returned")
	}

	err = img.checkMagic()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, "image/jpeg", img.mime, "Mime type should be the same")
	}

}

func TestCheckMagicBad(t *testing.T) {

	req := formRandomRequest("test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err := img.copyBytes()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.image, "File bytes should be returned")
	}

	err = img.getHash()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img.SHA, "SHA should be returned")
	}

	err = img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("unknown file type"), err, "Error should match")
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
		assert.Contains(t, fmt.Sprintf("%s", err), "image filesize too large", "Error should match")
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
		assert.Contains(t, fmt.Sprintf("%s", err), "image width too small", "Error should match")
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
		assert.Contains(t, fmt.Sprintf("%s", err), "image width too large", "Error should match")
	}

}

func TestMakeFilenames(t *testing.T) {

	img := ImageType{}

	img.makeFilenames()

	assert.NotEmpty(t, img.Filename, "Filename should be returned")

	assert.NotEmpty(t, img.Thumbnail, "Thumbnail name should be returned")

}

func TestSaveFile(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	noban := sqlmock.NewRows([]string{"count"}).AddRow(0)
	nodupe := sqlmock.NewRows([]string{"count", "post", "thread"}).AddRow(0, 0, 0)

	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_files WHERE ban_hash`).WillReturnRows(noban)
	mock.ExpectQuery(`select count\(1\),posts.post_num,threads.thread_id from threads`).WillReturnRows(nodupe)

	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	config.Settings.Limits.ImageMaxSize = 3000000

	req := formJpegRequest(1000, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	img.Ib = 1

	err = img.SaveImage()
	if assert.NoError(t, err, "An error was not expected") {
		assert.NotEmpty(t, img.MD5, "MD5 should be returned")
		assert.NotEmpty(t, img.SHA, "SHA should be returned")
		assert.Equal(t, ".jpg", img.Ext, "Ext should be the same")
		assert.Equal(t, "image/jpeg", img.mime, "Mime type should be the same")
		assert.Equal(t, 1000, img.OrigHeight, "Height should be the same")
		assert.Equal(t, 1000, img.OrigWidth, "Width should be the same")
		assert.NotZero(t, img.ThumbHeight, "Thumbnail height should be returned")
		assert.NotZero(t, img.ThumbWidth, "Thumbnail width should be returned")
		assert.NotEmpty(t, img.Filename, "Filename should be returned")
		assert.NotEmpty(t, img.Thumbnail, "Thumbnail name should be returned")
	}

	file, err := os.Open(img.Filepath)
	assert.NoError(t, err, "An error was not expected")

	fileinfo, err := file.Stat()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, img.Filename, fileinfo.Name(), "Name should be the same")
	}

	thumb, err := os.Open(img.Thumbpath)
	assert.NoError(t, err, "An error was not expected")

	thumbinfo, err := thumb.Stat()
	if assert.NoError(t, err, "An error was not expected") {
		assert.Equal(t, img.Thumbnail, thumbinfo.Name(), "Name should be the same")
	}

}

func TestSaveFileNoIb(t *testing.T) {

	var err error

	mock, err := db.NewTestDb()
	assert.NoError(t, err, "An error was not expected")
	defer db.CloseDb()

	noban := sqlmock.NewRows([]string{"count"}).AddRow(0)
	nodupe := sqlmock.NewRows([]string{"count", "post", "thread"}).AddRow(0, 0, 0)

	mock.ExpectQuery(`SELECT count\(\*\) FROM banned_files WHERE ban_hash`).WillReturnRows(noban)
	mock.ExpectQuery(`select count\(1\),posts.post_num,threads.thread_id from threads`).WillReturnRows(nodupe)

	req := formJpegRequest(300, "test.jpeg")

	img := ImageType{}

	img.File, img.Header, _ = req.FormFile("file")

	err = img.SaveImage()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("no imageboard set on duplicate check"), err, "Error should match")
	}

}
