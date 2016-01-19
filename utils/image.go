package utils

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eirka/eirka-libs/amazon"
	"github.com/eirka/eirka-libs/config"

	local "github.com/eirka/eirka-post/config"
)

// valid file extensions
var validExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webm": true,
}

type ImageType struct {
	File        multipart.File
	Header      *multipart.FileHeader
	Filename    string
	Thumbnail   string
	Ext         string
	MD5         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
	image       []byte
	mime        string
	duration    int
}

// ProcessFile will check file integrity, get an md5, and make filenames
func (i *ImageType) ProcessFile() (err error) {

	// check given file ext
	err = i.checkReqExt()
	if err != nil {
		return
	}

	// get file md5
	err = i.getMD5()
	if err != nil {
		return
	}

	// check file magic sig
	err = i.checkMagic()
	if err != nil {
		return
	}

	return

}

// save an image file
func (i *ImageType) SaveImage() (err error) {

	// check image stats
	err = i.getStats()
	if err != nil {
		return
	}

	// create unique filenames for image and thumbnail
	err = i.makeFilenames()
	if err != nil {
		return
	}

	// save the file to disk
	err = i.saveFile()
	if err != nil {
		return
	}

	// create a thumbnail
	err = i.createThumbnail()
	if err != nil {
		return
	}

	return

}

// Get file extension from request header
func (i *ImageType) checkReqExt() (err error) {
	// Get ext from request header
	name := i.Header.Filename
	ext := filepath.Ext(name)

	if ext == "" {
		return errors.New("no file extension")
	}

	// Check to see if extension is allowed
	ext_check := isAllowedExt(ext)
	if !ext_check {
		return errors.New("format not supported")
	}

	return

}

// Check if file ext allowed
func isAllowedExt(ext string) bool {

	if validExt[strings.ToLower(ext)] {
		return true
	}

	return false

}

// Get image MD5 and write file into buffer
func (i *ImageType) getMD5() (err error) {
	var b bytes.Buffer

	hasher := md5.New()

	buffer := bufio.NewWriter(&b)

	// Save file and also read into hasher for md5
	_, err = io.Copy(buffer, io.TeeReader(i.File, hasher))
	if err != nil {
		return errors.New("problem copying file")
	}

	err = buffer.Flush()
	if err != nil {
		return errors.New("problem copying file")
	}

	// Set md5sum from hasher
	i.MD5 = hex.EncodeToString(hasher.Sum(nil))

	i.image = b.Bytes()

	err = i.File.Close()
	if err != nil {
		return errors.New("problem copying file")
	}

	return

}

func (i *ImageType) checkMagic() (err error) {

	i.mime = http.DetectContentType(i.image)

	switch i.mime {
	case "image/png":
		i.Ext = ".png"
	case "image/jpeg":
		i.Ext = ".jpg"
	case "image/gif":
		i.Ext = ".gif"
	case "video/webm":
		i.Ext = ".webm"
	default:
		return errors.New("unknown file type")
	}

	// Check to see if extension is allowed
	ext_check := isAllowedExt(i.Ext)
	if !ext_check {
		return errors.New("format not supported")
	}

	return

}

func (i *ImageType) getStats() (err error) {
	buffer := bytes.NewReader(i.image)

	imagesize := buffer.Len()

	img, _, err := image.DecodeConfig(buffer)
	if err != nil {
		return errors.New("problem decoding image")
	}

	i.OrigWidth = img.Width
	i.OrigHeight = img.Height

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return errors.New("image width too large")
	case img.Width < config.Settings.Limits.ImageMinWidth:
		return errors.New("image width too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return errors.New("image height too large")
	case img.Height < config.Settings.Limits.ImageMinHeight:
		return errors.New("image height too small")
	case imagesize > config.Settings.Limits.ImageMaxSize:
		return errors.New("image size too large")
	}

	return

}

// Make a random unix time filename
func (i *ImageType) makeFilenames() (err error) {

	// Create seed for random
	rand.Seed(time.Now().UnixNano())
	// Get random 3 digit int to append to unix time
	rand_t := rand.Intn(899) + 100
	// Get current unix time
	time_t := time.Now().Unix()
	// Append random int to unix time
	file_t := fmt.Sprintf("%d%d", time_t, rand_t)
	// Append ext to filename
	i.Filename = fmt.Sprintf("%s%s", file_t, i.Ext)
	// Append jpg to thumbnail name because it is always a jpg
	i.Thumbnail = fmt.Sprintf("%s%s%s", file_t, "s", ".jpg")

	return

}

func (i *ImageType) saveFile() (err error) {
	buffer := bytes.NewReader(i.image)

	imagefile := filepath.Join(local.Settings.Directories.ImageDir, i.Filename)

	image, err := os.Create(imagefile)
	if err != nil {
		return errors.New("problem saving file")
	}
	defer image.Close()

	_, err = io.Copy(image, buffer)
	if err != nil {
		return errors.New("problem saving file")
	}

	s3 := amazon.New()

	err = s3.Save(imagefile, fmt.Sprintf("src/%s", i.Filename), i.mime)
	if err != nil {
		return
	}

	return

}

func (i *ImageType) createThumbnail() (err error) {

	object := amazon.LambdaThumbnail{
		Bucket:    config.Settings.Amazon.Bucket,
		Filename:  i.Filename,
		Thumbnail: i.Thumbnail,
		MaxWidth:  config.Settings.Limits.ThumbnailMaxWidth,
		MaxHeight: config.Settings.Limits.ThumbnailMaxHeight,
	}

	lambda := amazon.New()

	// run our lambda job and get the dimensions
	i.ThumbWidth, i.ThumbHeight, err = lambda.Execute(object)
	if err != nil {
		return
	}

	return

}
