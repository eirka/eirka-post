package utils

import (
	"bytes"
	"crypto/md5"
	"database/sql"
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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eirka/eirka-libs/amazon"
	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	_ "github.com/eirka/eirka-libs/errors"

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
	Ib          uint
	Filename    string
	Thumbnail   string
	Filepath    string
	Thumbpath   string
	Ext         string
	MD5         string
	OrigWidth   int
	OrigHeight  int
	ThumbWidth  int
	ThumbHeight int
	image       *bytes.Buffer
	mime        string
	duration    int
	video       bool
	avatar      bool
}

func (i *ImageType) IsValid() bool {

	if i.Filename == "" {
		return false
	}

	if i.Filepath == "" {
		return false
	}

	if i.Thumbnail == "" {
		return false
	}

	if i.Thumbpath == "" {
		return false
	}

	if i.Ib == 0 {
		return false
	}

	if i.Ext == "" {
		return false
	}

	if i.MD5 == "" {
		return false
	}

	if i.mime == "" {
		return false
	}

	return true
}

func (i *ImageType) IsValidPost() bool {
	if i.OrigWidth == 0 {
		return false
	}

	if i.OrigHeight == 0 {
		return false
	}

	if i.ThumbWidth == 0 {
		return false
	}

	if i.ThumbHeight == 0 {
		return false
	}

	return true
}

// save an image file
func (i *ImageType) SaveImage() (err error) {

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

	// check to see if the file already exists
	err = i.checkDuplicate()
	if err != nil {
		return
	}

	// check file magic sig
	err = i.checkMagic()
	if err != nil {
		return
	}

	// check image stats
	err = i.getStats()
	if err != nil {
		return
	}

	// save the file to disk
	err = i.saveFile()
	if err != nil {
		return
	}

	// process a webm
	if i.video {
		// check the webm info
		err = i.checkWebM()
		if err != nil {
			return
		}

		// create thumbnail from webm
		err = i.createWebMThumbnail()
		if err != nil {
			return
		}

	}

	// create a thumbnail
	err = i.createThumbnail(config.Settings.Limits.ThumbnailMaxWidth, config.Settings.Limits.ThumbnailMaxHeight)
	if err != nil {
		return
	}

	// copy the file to s3
	err = i.copyToS3()
	if err != nil {
		return
	}

	// check final state
	if !i.IsValidPost() {
		return errors.New("ImageType is not valid")
	}

	return

}

// Get file extension from request header
func (i *ImageType) checkReqExt() (err error) {
	// Get ext from request header
	name := i.Header.Filename
	ext := filepath.Ext(name)

	if ext == "" {
		return errors.New("No file extension")
	}

	// Check to see if extension is allowed
	if !isAllowedExt(ext) {
		return errors.New("Format not supported")
	}

	return

}

// Check if file ext allowed
func isAllowedExt(ext string) bool {
	return validExt[strings.ToLower(ext)]
}

// Get image MD5 and write file into buffer
func (i *ImageType) getMD5() (err error) {

	defer i.File.Close()

	hasher := md5.New()

	i.image = new(bytes.Buffer)

	// Save file and also read into hasher for md5
	_, err = io.Copy(i.image, io.TeeReader(i.File, hasher))
	if err != nil {
		return errors.New("Problem copying file")
	}

	// Set md5sum from hasher
	i.MD5 = hex.EncodeToString(hasher.Sum(nil))

	return

}

// check if the md5 is already in the database
func (i *ImageType) checkDuplicate() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	if i.Ib == 0 {
		return errors.New("No imageboard set on duplicate check")
	}

	var check bool
	var thread, post sql.NullInt64

	err = dbase.QueryRow(`select count(1),posts.post_num,threads.thread_id from threads 
	LEFT JOIN posts on threads.thread_id = posts.thread_id 
	LEFT JOIN images on posts.post_id = images.post_id 
	WHERE image_hash = ? AND ib_id = ?`, i.MD5, i.Ib).Scan(&check, &post, &thread)
	if err != nil {
		return
	}

	// return error if it exists
	if check {
		return fmt.Errorf("Image has already been posted. Thread: %d Post: %d", thread.Int64, post.Int64)
	}

	return
}

func (i *ImageType) checkMagic() (err error) {

	// detect the mime type
	i.mime = http.DetectContentType(i.image.Bytes())

	switch i.mime {
	case "image/png":
		i.Ext = ".png"
	case "image/jpeg":
		i.Ext = ".jpg"
	case "image/gif":
		i.Ext = ".gif"
	case "video/webm":
		i.Ext = ".webm"
		i.video = true
	default:
		return errors.New("Unknown file type")
	}

	// Check to see if extension is allowed
	if !isAllowedExt(i.Ext) {
		return errors.New("Format not supported")
	}

	return

}

func (i *ImageType) getStats() (err error) {

	// skip if its a video since we cant decode it
	if i.video {
		return
	}

	// decode image config
	img, _, err := image.DecodeConfig(bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("Problem decoding image")
	}

	// set original width
	i.OrigWidth = img.Width
	// set original height
	i.OrigHeight = img.Height

	// Check against maximum sizes
	switch {
	case i.OrigWidth > config.Settings.Limits.ImageMaxWidth:
		return errors.New("Image width too large")
	case img.Width < config.Settings.Limits.ImageMinWidth:
		return errors.New("Image width too small")
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return errors.New("Image height too large")
	case img.Height < config.Settings.Limits.ImageMinHeight:
		return errors.New("Image height too small")
	case i.image.Len() > config.Settings.Limits.ImageMaxSize:
		return errors.New("Image size too large")
	}

	return

}

func (i *ImageType) saveFile() (err error) {

	defer i.image.Reset()

	i.makeFilenames()

	// avatar filename is the users id
	if i.avatar {
		i.Thumbnail = fmt.Sprintf("%d.png", i.Ib)
		i.Thumbpath = filepath.Join(local.Settings.Directories.ThumbnailDir, i.Thumbnail)
	}

	if !i.IsValid() {
		return errors.New("ImageType is not valid")
	}

	imagefile := filepath.Join(local.Settings.Directories.ImageDir, i.Filename)

	image, err := os.Create(imagefile)
	if err != nil {
		return errors.New("Problem saving file")
	}
	defer image.Close()

	_, err = io.Copy(image, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("Problem saving file")
	}

	return

}

// Make a random unix time filename
func (i *ImageType) makeFilenames() {

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
	i.Thumbnail = fmt.Sprintf("%ss.jpg", file_t)

	// set the full file path
	i.Filepath = filepath.Join(local.Settings.Directories.ImageDir, i.Filename)

	// set the full thumbnail path
	i.Thumbpath = filepath.Join(local.Settings.Directories.ThumbnailDir, i.Thumbnail)

	return

}

func (i *ImageType) createThumbnail(maxwidth, maxheight int) (err error) {

	var imagef string

	if i.video {
		imagef = fmt.Sprintf("%s[0]", i.Thumbpath)
	} else {
		imagef = fmt.Sprintf("%s[0]", i.Filepath)
	}

	orig_dimensions := fmt.Sprintf("%dx%d", i.OrigWidth, i.OrigHeight)

	var args []string

	// different options for avatars
	if i.avatar {
		args = []string{
			"-size",
			orig_dimensions,
			imagef,
			"-background",
			"none",
			"-thumbnail",
			fmt.Sprintf("%dx%d^", maxwidth, maxheight),
			"-gravity",
			"center",
			"-extent",
			fmt.Sprintf("%dx%d", maxwidth, maxheight),
			i.Thumbpath,
		}
	} else {
		args = []string{
			"-background",
			"white",
			"-flatten",
			"-size",
			orig_dimensions,
			"-resize",
			fmt.Sprintf("%dx%d>", maxwidth, maxheight),
			"-quality",
			"90",
			imagef,
			i.Thumbpath,
		}
	}

	_, err = exec.Command("convert", args...).Output()
	if err != nil {
		return errors.New("Problem making thumbnail")
	}

	thumb, err := os.Open(i.Thumbpath)
	if err != nil {
		return errors.New("Problem making thumbnail")
	}
	defer thumb.Close()

	img, _, err := image.DecodeConfig(thumb)
	if err != nil {
		return errors.New("Problem decoding thumbnail")
	}

	i.ThumbWidth = img.Width
	i.ThumbHeight = img.Height

	return

}

func (i *ImageType) copyToS3() (err error) {

	s3 := amazon.New()

	err = s3.Save(i.Filepath, fmt.Sprintf("src/%s", i.Filename), i.mime, false)
	if err != nil {
		return
	}

	err = s3.Save(i.Thumbpath, fmt.Sprintf("thumb/%s", i.Thumbnail), "image/jpeg", false)
	if err != nil {
		return
	}

	return
}
