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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/techjanitor/pram-post/config"
)

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
}

// Get file extension from request header
func (i *ImageType) CheckReqExt() (err error) {
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

	validExt := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webm": true,
	}

	if validExt[strings.ToLower(ext)] {
		return true
	}

	return false

}

// Make a random unix time filename
func (i *ImageType) MakeFilenames() (err error) {

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

// Get image MD5 and write file into buffer
func (i *ImageType) GetMD5() (err error) {
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

func (i *ImageType) CheckMagic() (err error) {
	buffer := bytes.NewReader(i.image)

	bytes := make([]byte, 4)

	n, _ := buffer.ReadAt(bytes, 0)

	if n < 4 {
		return errors.New("unknown file type")
		// PNG Signature
	} else if bytes[0] == 0x89 && bytes[1] == 0x50 && bytes[2] == 0x4E && bytes[3] == 0x47 {
		i.Ext = ".png"
		// JPG Signature
	} else if bytes[0] == 0xFF && bytes[1] == 0xD8 && bytes[2] == 0xFF {
		i.Ext = ".jpg"
		// Gif Signature
	} else if bytes[0] == 0x47 && bytes[1] == 0x49 && bytes[2] == 0x46 && bytes[3] == 0x38 {
		i.Ext = ".gif"
		// WebM Signature
	} else if bytes[0] == 0x1A && bytes[1] == 0x45 && bytes[2] == 0xDF && bytes[3] == 0xA3 {
		i.Ext = ".webm"
	} else {
		return errors.New("unknown file type")
	}

	// Check to see if extension is allowed
	ext_check := isAllowedExt(i.Ext)
	if !ext_check {
		return errors.New("format not supported")
	}

	return

}

func (i *ImageType) GetStats() (err error) {
	buffer := bytes.NewReader(i.image)

	imagesize := buffer.Len()

	img, _, err := image.DecodeConfig(buffer)
	if err != nil {
		return errors.New("problem decoding image")
	}

	i.OrigWidth = img.Width
	i.OrigHeight = img.Height

	// Check against maximum sizes
	if i.OrigWidth > config.Settings.Limits.ImageMaxWidth {
		return errors.New("image width too large")
	} else if img.Width < config.Settings.Limits.ImageMinWidth {
		return errors.New("image width too small")
	} else if i.OrigHeight > config.Settings.Limits.ImageMaxHeight {
		return errors.New("image height too large")
	} else if img.Height < config.Settings.Limits.ImageMinHeight {
		return errors.New("image height too small")
	} else if imagesize > config.Settings.Limits.ImageMaxSize {
		return errors.New("image size too large")
	}

	return

}

func (i *ImageType) SaveImage() (err error) {
	buffer := bytes.NewReader(i.image)

	imagefile := filepath.Join(config.Settings.General.ImageDir, i.Filename)

	image, err := os.Create(imagefile)
	if err != nil {
		os.RemoveAll(imagefile)
		return errors.New("problem creating file")
	}
	defer image.Close()

	_, err = io.Copy(image, buffer)
	if err != nil {
		os.RemoveAll(imagefile)
		return errors.New("problem creating file")
	}

	return

}

func (i *ImageType) CreateThumbnail() (err error) {
	imagefile := filepath.Join(config.Settings.General.ImageDir, i.Filename)
	thumbfile := filepath.Join(config.Settings.General.ThumbnailDir, i.Thumbnail)

	orig_dimensions := fmt.Sprintf("%dx%d", i.OrigWidth, i.OrigHeight)
	thumb_dimensions := fmt.Sprintf("%dx%d>", config.Settings.Limits.ThumbnailMaxWidth, config.Settings.Limits.ThumbnailMaxHeight)
	imagef := fmt.Sprintf("%s[0]", imagefile)

	args := []string{
		"-background",
		"white",
		"-flatten",
		"-size",
		orig_dimensions,
		"-resize",
		thumb_dimensions,
		"-quality",
		"90",
		imagef,
		thumbfile,
	}

	_, err = exec.Command("convert", args...).Output()
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem making thumbnail")
	}

	thumb, err := os.Open(thumbfile)
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem making thumbnail")
	}
	defer thumb.Close()

	img, _, err := image.DecodeConfig(thumb)
	if err != nil {
		os.RemoveAll(thumbfile)
		os.RemoveAll(imagefile)
		return errors.New("problem decoding thumbnail")
	}

	i.ThumbWidth = img.Width
	i.ThumbHeight = img.Height

	return

}
