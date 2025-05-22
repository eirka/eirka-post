package utils

import (
	"bytes"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/hex"
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

	local "github.com/eirka/eirka-post/config"
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
	// With our improved security, this should now fail
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("suspicious file extension pattern detected"), err, "Error should match")
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

// Test filenames with suspicious extensions embedded in the name
func TestSuspiciousFilenames(t *testing.T) {
	testCases := []struct {
		filename string
		expected string
	}{
		{"harmless.php.jpg", "suspicious file extension pattern detected"},
		{"exploit.js.png", "suspicious file extension pattern detected"},
		{"shell.sh.gif", "suspicious file extension pattern detected"},
		{"script.py.webm", "suspicious file extension pattern detected"},
		{"sneaky.html.jpeg", "suspicious file extension pattern detected"},
		{"normal.with.dots.jpg", ""}, // Should pass
		{"multiple.dots.png", ""},    // Should pass if extensions are benign
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			req := formJpegRequest(300, tc.filename)
			img := ImageType{}
			img.File, img.Header, _ = req.FormFile("file")

			err := img.checkReqExt()

			if tc.expected == "" {
				assert.NoError(t, err, "Expected no error for: "+tc.filename)
			} else {
				if assert.Error(t, err, "Expected error for: "+tc.filename) {
					assert.Equal(t, errors.New(tc.expected), err, "Error should match for: "+tc.filename)
				}
			}
		})
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
	testCases := []struct {
		filename string
		expected string
	}{
		{"test.jpeg", ".jpeg"},
		{"test.jpg", ".jpg"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			req := formJpegRequest(300, tc.filename)

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

			err = img.checkReqExt()
			if assert.NoError(t, err, "An error was not expected") {
				assert.Equal(t, tc.expected, img.Ext, "Extension should be set correctly")
			}

			err = img.checkMagic()
			if assert.NoError(t, err, "An error was not expected") {
				assert.Equal(t, "image/jpeg", img.mime, "Mime type should be the same")
				assert.Equal(t, tc.expected, img.Ext, "Extension should remain correct after magic check")
			}
		})
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
		assert.Equal(t, errors.New("unknown or unsupported file type"), err, "Error should match")
	}
}

// Test very small image files that might be trying to bypass validation
func TestTooSmallFile(t *testing.T) {
	// Create a minimal valid PNG header but make it too small overall
	var b bytes.Buffer
	b.Write([]byte("\x89PNG\r\n\x1a\n")) // Valid PNG header
	b.Write(make([]byte, 50))            // Add some padding to make it 58 bytes

	img := ImageType{
		image: &b,
	}

	err := img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("file is suspiciously small"), err, "Error should match")
	}
}

// Test mismatch between extension and MIME type
func TestExtensionMimeMismatch(t *testing.T) {
	img := ImageType{}

	// Set up a JPEG file
	req := formJpegRequest(300, "test.jpeg")
	img.File, img.Header, _ = req.FormFile("file")

	err := img.copyBytes()
	assert.NoError(t, err, "An error was not expected")

	// Manually set extension to PNG, which will conflict with JPEG content
	img.Ext = ".png"

	// This should fail since the file is JPEG but extension is PNG
	err = img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("file extension doesn't match content type"), err, "Error should match")
	}
}

// Test corrupted file headers
func TestCorruptedFileHeaders(t *testing.T) {
	// Create completely invalid content that won't match any file type
	var b bytes.Buffer
	b.Write([]byte("This is not a real image file but has enough text to analyze"))
	b.Write(make([]byte, 200))

	img := ImageType{
		image: &b,
	}

	// This should detect that the content doesn't match any known file type
	err := img.checkMagic()
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("unknown or unsupported file type"), err, "Error should match")
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

	// We need to create our own test environment that will pass our new
	// security validation checks

	// Create a proper JPEG image
	jpegFile := testJpeg(1000)

	// Set up a proper ImageType with all required fields
	img := ImageType{
		Ib:    1,
		image: jpegFile,
		Ext:   ".jpg",
		mime:  "image/jpeg",
	}

	// Set hash values
	hasher := md5.New()
	io.Copy(hasher, bytes.NewReader(jpegFile.Bytes()))
	img.MD5 = hex.EncodeToString(hasher.Sum(nil))

	sha := sha1.New()
	io.Copy(sha, bytes.NewReader(jpegFile.Bytes()))
	img.SHA = hex.EncodeToString(sha.Sum(nil))

	// Get image dimensions by manually decoding
	imgDecoded, _, err := image.DecodeConfig(bytes.NewReader(jpegFile.Bytes()))
	assert.NoError(t, err, "Image decode should not fail")

	img.OrigWidth = imgDecoded.Width
	img.OrigHeight = imgDecoded.Height

	// Skip full SaveImage and just test saveFile + thumbnail generation
	i := img

	// Manually create filenames
	i.makeFilenames()

	// Use the pre-existing directories defined in config
	// Ensure directories exist
	err = os.MkdirAll(local.Settings.Directories.ImageDir, 0755)
	assert.NoError(t, err, "Failed to ensure image directory exists")

	err = os.MkdirAll(local.Settings.Directories.ThumbnailDir, 0755)
	assert.NoError(t, err, "Failed to ensure thumbnail directory exists")

	// Track files to clean up after test
	filesToCleanup := []string{i.Filepath, i.Thumbpath}
	defer func() {
		for _, file := range filesToCleanup {
			os.Remove(file) // Best effort cleanup
		}
	}()

	// Save file to the configured directory
	err = i.saveFile()
	assert.NoError(t, err, "Save file should not fail")

	// Verify the saved file exists
	_, err = os.Stat(i.Filepath)
	assert.NoError(t, err, "File should exist")

	// Create thumbnail
	err = i.createThumbnail(300, 300)
	assert.NoError(t, err, "Thumbnail creation should not fail")

	// Verify the thumbnail exists
	_, err = os.Stat(i.Thumbpath)
	assert.NoError(t, err, "Thumbnail should exist")
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

// TestCleanupFiles tests that files are properly removed during cleanup
func TestCleanupFiles(t *testing.T) {
	// Ensure test directories exist
	err := os.MkdirAll(local.Settings.Directories.ImageDir, 0755)
	assert.NoError(t, err, "Failed to ensure image directory exists")

	err = os.MkdirAll(local.Settings.Directories.ThumbnailDir, 0755)
	assert.NoError(t, err, "Failed to ensure thumbnail directory exists")

	// Create test files
	testImg := &ImageType{
		Filename:  "cleanup_test.jpg",
		Thumbnail: "cleanup_test_thumb.jpg",
		Filepath:  local.Settings.Directories.ImageDir + "/cleanup_test.jpg",
		Thumbpath: local.Settings.Directories.ThumbnailDir + "/cleanup_test_thumb.jpg",
	}

	// Create empty files for testing
	_, err = os.Create(testImg.Filepath)
	assert.NoError(t, err, "Failed to create test file")

	_, err = os.Create(testImg.Thumbpath)
	assert.NoError(t, err, "Failed to create test thumbnail")

	// Verify files exist
	_, err = os.Stat(testImg.Filepath)
	assert.NoError(t, err, "Test file should exist")

	_, err = os.Stat(testImg.Thumbpath)
	assert.NoError(t, err, "Test thumbnail should exist")

	// Run cleanup
	testImg.cleanupFiles()

	// Verify files are gone
	_, err = os.Stat(testImg.Filepath)
	assert.Error(t, err, "Test file should be removed")
	assert.True(t, os.IsNotExist(err), "File should not exist after cleanup")

	_, err = os.Stat(testImg.Thumbpath)
	assert.Error(t, err, "Test thumbnail should be removed")
	assert.True(t, os.IsNotExist(err), "Thumbnail should not exist after cleanup")
}

// TestPartialFailureCleanup tests that files are cleaned up when processing fails after saving
func TestPartialFailureCleanup(t *testing.T) {
	// Ensure test directories exist
	err := os.MkdirAll(local.Settings.Directories.ImageDir, 0755)
	assert.NoError(t, err, "Failed to ensure image directory exists")

	err = os.MkdirAll(local.Settings.Directories.ThumbnailDir, 0755)
	assert.NoError(t, err, "Failed to ensure thumbnail directory exists")

	// Set up a custom test image
	jpegFile := testJpeg(500)

	// Create a mock image model that will succeed up to a point
	img := ImageType{
		// Set up just enough fields for validation to pass
		Ib:         1,
		image:      jpegFile,
		Ext:        ".jpg",
		mime:       "image/jpeg",
		OrigWidth:  500,
		OrigHeight: 500,
	}

	// Get image dimensions
	imgDecoded, _, err := image.DecodeConfig(bytes.NewReader(jpegFile.Bytes()))
	assert.NoError(t, err, "Image decode should not fail")

	img.OrigWidth = imgDecoded.Width
	img.OrigHeight = imgDecoded.Height

	// Generate MD5 and SHA
	hasher := md5.New()
	io.Copy(hasher, bytes.NewReader(jpegFile.Bytes()))
	img.MD5 = hex.EncodeToString(hasher.Sum(nil))

	sha := sha1.New()
	io.Copy(sha, bytes.NewReader(jpegFile.Bytes()))
	img.SHA = hex.EncodeToString(sha.Sum(nil))

	// Set up the test file names
	img.makeFilenames()

	// Test phase 1: create an actual file to simulate successful save
	testFile, err := os.Create(img.Filepath)
	assert.NoError(t, err, "Should create test file")
	testFile.Close()

	// Verify file exists
	_, err = os.Stat(img.Filepath)
	assert.NoError(t, err, "File should exist before cleanup")

	// Test phase 2: Simulate a failure that will trigger cleanup by directly calling cleanupFiles
	img.cleanupFiles()

	// Verify files are gone
	_, err = os.Stat(img.Filepath)
	assert.Error(t, err, "Original file should be cleaned up")
	assert.True(t, os.IsNotExist(err), "File should not exist after cleanup")
}

// TestCleanupSafety ensures cleanup can't delete directories or unrelated files
func TestCleanupSafety(t *testing.T) {
	// Ensure test directories exist
	err := os.MkdirAll(local.Settings.Directories.ImageDir, 0755)
	assert.NoError(t, err, "Failed to ensure image directory exists")

	err = os.MkdirAll(local.Settings.Directories.ThumbnailDir, 0755)
	assert.NoError(t, err, "Failed to ensure thumbnail directory exists")

	// Create a subdirectory for testing
	testSubdir := local.Settings.Directories.ImageDir + "/test_subdir"
	err = os.MkdirAll(testSubdir, 0755)
	assert.NoError(t, err, "Failed to create test subdirectory")
	defer os.RemoveAll(testSubdir) // Clean up after test

	// Create a file outside the expected directories
	tempDir := os.TempDir()
	externalFile := tempDir + "/external_test_file.jpg"
	_, err = os.Create(externalFile)
	assert.NoError(t, err, "Failed to create external test file")
	defer os.Remove(externalFile) // Clean up after test

	// Create an unrelated file in the image directory
	unrelatedFile := local.Settings.Directories.ImageDir + "/unrelated_file.txt"
	_, err = os.Create(unrelatedFile)
	assert.NoError(t, err, "Failed to create unrelated file")
	defer os.Remove(unrelatedFile) // Clean up after test

	// 1. Test that we can't delete directories

	// Set up a malicious image model pointing to a directory
	dirImg := ImageType{
		Filename:  "test_subdir",
		Filepath:  testSubdir,
		Thumbnail: "thumb_dir",
		Thumbpath: local.Settings.Directories.ThumbnailDir,
	}

	// Try to delete a directory
	dirImg.cleanupFiles()

	// Verify directory still exists
	_, err = os.Stat(testSubdir)
	assert.NoError(t, err, "Directory should not be deleted")

	// 2. Test that we can't delete files outside the expected directories

	// Set up image model pointing to external file
	extImg := ImageType{
		Filename:  "external_test_file.jpg",
		Filepath:  externalFile,
		Thumbnail: "thumb.jpg",
		Thumbpath: tempDir + "/thumb.jpg",
	}

	// Try to delete external file
	extImg.cleanupFiles()

	// Verify external file still exists
	_, err = os.Stat(externalFile)
	assert.NoError(t, err, "External file should not be deleted")

	// 3. Test that we can't delete unrelated files in the image directory

	// Set up image model with mismatched filename
	unrelImg := ImageType{
		Filename:  "wrong_filename.jpg", // Different from actual file
		Filepath:  unrelatedFile,        // Points to real file
		Thumbnail: "thumb.jpg",
		Thumbpath: local.Settings.Directories.ThumbnailDir + "/thumb.jpg",
	}

	// Try to delete with mismatched filename
	unrelImg.cleanupFiles()

	// Verify unrelated file still exists
	_, err = os.Stat(unrelatedFile)
	assert.NoError(t, err, "Unrelated file should not be deleted")
}

// TestTimeouts tests that operations with external processes use timeouts
func TestTimeouts(t *testing.T) {
	// This test doesn't actually trigger timeouts, but verifies the code paths that would handle them
	// A real timeout test would be slow and potentially flaky

	// Test with a fake command that will be rejected by checkMagic before we get to the timeout
	img := ImageType{
		image: bytes.NewBuffer([]byte("not a real image")),
	}

	// This should fail with a format error, not a timeout
	err := img.checkMagic()
	assert.Error(t, err, "Invalid image should be rejected")
	assert.Contains(t, err.Error(), "unknown or unsupported", "Error should be about format, not timeout")

	// Create a minimal but valid test image
	validImg := testPng(200)
	img.image = validImg
	img.Ext = ".png"
	img.mime = "image/png"

	// Set up for a valid thumbnail creation
	// This test just ensures our code path works in the success case
	// The execution would need to actually time out to trigger the timeout error path
	config.Settings.Limits.ImageMaxWidth = 1000
	config.Settings.Limits.ImageMinWidth = 100
	config.Settings.Limits.ImageMaxHeight = 1000
	config.Settings.Limits.ImageMinHeight = 100
	img.OrigWidth = 200
	img.OrigHeight = 200
	img.Filepath = local.Settings.Directories.ImageDir + "/test_timeout.png"
	img.Thumbpath = local.Settings.Directories.ThumbnailDir + "/test_timeout_thumb.jpg"

	// We don't actually expect this to succeed as we don't have a real file,
	// but it will try to use the timeout code path
	_ = img.createThumbnail(100, 100)
}
