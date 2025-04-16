package utils

import (
	"bytes"
	"errors"

	"github.com/1l0/identicon"
)

// SaveAvatar will save a user provided avatar
func (i *ImageType) SaveAvatar() (err error) {

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

	// check file magic sig
	err = i.checkMagic()
	if err != nil {
		return
	}

	// videos cant be avatars
	if i.video {
		return errors.New("format not supported")
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

	return
}
