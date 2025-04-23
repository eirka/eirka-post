package utils

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/1l0/identicon"
)

// SaveAvatar will save a user provided avatar
func (i *ImageType) SaveAvatar() (err error) {
	// Successful completion flag
	var success bool
	// Defer cleanup on failure
	defer func() {
		// If we didn't complete successfully, clean up any created files
		if !success {
			i.cleanupFiles()
		}
	}()

	// for special handling
	i.avatar = true

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

	// get file hash
	err = i.getHash()
	if err != nil {
		return
	}

	// For test compatibility, we need to set mime type and extension before checkMagic
	// because our new validation checks extension/mime type consistency
	i.mime = http.DetectContentType(i.image.Bytes())

	// check file magic sig - with more advanced validation
	err = i.checkMagic()
	if err != nil {
		return
	}

	// videos cant be avatars
	if i.video {
		err = errors.New("format not supported")
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

	// create a thumbnail
	err = i.createThumbnail(128, 128)
	if err != nil {
		return
	}

	// Mark as successful to prevent cleanup
	success = true
	return
}

// GenerateAvatar this will create a random avatar
func GenerateAvatar(uid uint) (err error) {
	if uid == 0 || uid == 1 {
		return errors.New("invalid user id")
	}

	img := ImageType{
		avatar:     true,
		OrigWidth:  420,
		OrigHeight: 420,
		Ext:        ".png",
		image:      new(bytes.Buffer),
		Ib:         uid,
		MD5:        "fake",
		SHA:        "fake",
		mime:       "image/png",
	}

	// Successful completion flag
	var success bool
	// Defer cleanup on failure
	defer func() {
		// If we didn't complete successfully, clean up any created files
		if !success {
			img.cleanupFiles()
		}
	}()

	// generates a random avatar
	id := identicon.New()
	// a colorful theme
	id.Theme = identicon.Free
	// put the output into our image buffer
	err = id.GeneratePNG(img.image)
	if err != nil {
		return
	}

	// save the file to disk
	err = img.saveFile()
	if err != nil {
		return
	}

	// create a thumbnail
	err = img.createThumbnail(128, 128)
	if err != nil {
		return
	}

	// Mark as successful to prevent cleanup
	success = true
	return
}
