package utils

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"

	// gif support
	_ "image/gif"
	// jpeg support
	_ "image/jpeg"
	// png support
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	shortid "github.com/ventu-io/go-shortid"

	"github.com/eirka/eirka-libs/amazon"
	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"

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

func init() {

	var err error

	// test for ImageMagick
	_, err = exec.Command("convert", "--version").Output()
	if err != nil {
		panic("ImageMagick not found")
	}

}

// FileUploader defines the file processing functions
type FileUploader interface {
	// struct integrity
	IsValid() bool
	IsValidPost() bool

	// image processing
	SaveImage() (err error)
	checkReqExt() (err error)
	copyBytes() (err error)
	getHash() (err error)
	checkBanned() (err error)
	checkDuplicate() (err error)
	checkMagic() (err error)
	getStats() (err error)
	saveFile() (err error)
	makeFilenames()
	createThumbnail(maxwidth, maxheight int) (err error)
	copyToS3() (err error)

	// webm specific functions
	checkWebM() (err error)
	createWebMThumbnail() (err error)

	// avatar functions
	SaveAvatar() (err error)
	avatarToS3() (err error)
}

// ImageType defines an image and its metadata for processing
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
	SHA         string
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

var _ = FileUploader(&ImageType{})

// IsValid will check struct integrity
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

	if i.SHA == "" {
		return false
	}

	if i.mime == "" {
		return false
	}

	return true
}

// IsValidPost will check final struct integrity
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

// SaveImage runs the entire file processing pipeline
func (i *ImageType) SaveImage() (err error) {

	// check given file ext
	err = i.checkReqExt()
	if err != nil {
		return
	}

	// copy the multipart file into a buffer
	err = i.copyBytes()
	if err != nil {
		return
	}

	// get file md5
	err = i.getHash()
	if err != nil {
		return
	}

	// check to see if the file is banned
	err = i.checkBanned()
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

// Copy the multipart file into a bytes buffer
func (i *ImageType) copyBytes() (err error) {
	defer i.File.Close()

	i.image = new(bytes.Buffer)

	// Save file and also read into hasher for md5
	_, err = io.Copy(i.image, i.File)
	if err != nil {
		return errors.New("Problem copying file to buffer")
	}

	return
}

// Get image MD5 and write file into buffer
func (i *ImageType) getHash() (err error) {

	hasher := md5.New()

	sha := sha1.New()

	// read into hasher for md5
	_, err = io.Copy(hasher, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("Problem creating MD5 hash")
	}

	// Set md5sum from hasher
	i.MD5 = hex.EncodeToString(hasher.Sum(nil))

	// read into hasher for md5
	_, err = io.Copy(sha, bytes.NewReader(i.image.Bytes()))
	if err != nil {
		return errors.New("Problem creating SHA1 hash")
	}

	// Set sha1
	i.SHA = hex.EncodeToString(sha.Sum(nil))

	return

}

// check if the md5 is a banned file
func (i *ImageType) checkBanned() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	if i.MD5 == "" {
		return errors.New("No hash set on file banned check")
	}

	var check bool

	err = dbase.QueryRow(`SELECT count(*) FROM banned_files WHERE ban_hash = ?`, i.MD5).Scan(&check)
	if err != nil {
		return
	}

	// return error if it exists
	if check {
		return fmt.Errorf("File is banned")
	}

	return
}

// check if the md5 is already in the database
func (i *ImageType) checkDuplicate() (err error) {

	// Get Database handle
	dbase, err := db.GetDb()
	if err != nil {
		return
	}

	if i.MD5 == "" {
		return errors.New("No hash set on duplicate check")
	}

	if i.Ib == 0 {
		return errors.New("No imageboard set on duplicate check")
	}

	var check bool
	var thread, post sql.NullInt64

	err = dbase.QueryRow(`select count(1),posts.post_num,threads.thread_id from threads
	LEFT JOIN posts on threads.thread_id = posts.thread_id
	LEFT JOIN images on posts.post_id = images.post_id
	WHERE image_hash = ? AND ib_id = ? AND post_deleted = 0`, i.MD5, i.Ib).Scan(&check, &post, &thread)
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
		return fmt.Errorf("Image width too large. Max: %dpx", config.Settings.Limits.ImageMaxWidth)
	case img.Width < config.Settings.Limits.ImageMinWidth:
		return fmt.Errorf("Image width too small. Min: %dpx", config.Settings.Limits.ImageMinWidth)
	case i.OrigHeight > config.Settings.Limits.ImageMaxHeight:
		return fmt.Errorf("Image height too large. Max: %dpx", config.Settings.Limits.ImageMaxHeight)
	case img.Height < config.Settings.Limits.ImageMinHeight:
		return fmt.Errorf("Image height too small. Min: %dpx", config.Settings.Limits.ImageMinHeight)
	case i.image.Len() > config.Settings.Limits.ImageMaxSize:
		return fmt.Errorf("Image filesize too large. Max: %dMB", (config.Settings.Limits.ImageMaxSize/1024)/1024)
	}

	return

}

func (i *ImageType) saveFile() (err error) {

	defer i.image.Reset()

	i.makeFilenames()

	// avatar filename is the users id
	if i.avatar {
		i.Thumbnail = fmt.Sprintf("%d.png", i.Ib)
		i.Thumbpath = filepath.Join(local.Settings.Directories.AvatarDir, i.Thumbnail)
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

	// get new short id generator
	sid := shortid.MustNew(1, shortid.DefaultABC, 9001)

	// generate filename
	filename := sid.MustGenerate()

	// Append ext to filename
	i.Filename = fmt.Sprintf("%s%s", filename, i.Ext)

	// Append jpg to thumbnail name because it is always a jpg
	i.Thumbnail = fmt.Sprintf("%ss.jpg", filename)

	// set the full file path
	i.Filepath = filepath.Join(local.Settings.Directories.ImageDir, i.Filename)

	// set the full thumbnail path
	i.Thumbpath = filepath.Join(local.Settings.Directories.ThumbnailDir, i.Thumbnail)

}

func (i *ImageType) createThumbnail(maxwidth, maxheight int) (err error) {

	var imagef string

	if i.video {
		imagef = fmt.Sprintf("%s[0]", i.Thumbpath)
	} else {
		imagef = fmt.Sprintf("%s[0]", i.Filepath)
	}

	originalDimensions := fmt.Sprintf("%dx%d", i.OrigWidth, i.OrigHeight)

	var args []string

	// different options for avatars
	if i.avatar {
		args = []string{
			"-size",
			originalDimensions,
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
			originalDimensions,
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
		return errors.New("Problem creating thumbnail file")
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

	// noop if amazon is not configured
	if !config.Settings.Amazon.Configured {
		return
	}

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
