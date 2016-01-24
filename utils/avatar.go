package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/1l0/identicon"

	"github.com/eirka/eirka-libs/amazon"
)

// save an avatar
func (i *ImageType) SaveAvatar() (err error) {

	// for special handling
	i.avatar = true

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

	// videos cant be avatars
	if i.video {
		return errors.New("Format not supported")
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

	// copy the file to s3
	err = i.avatarToS3()
	if err != nil {
		return
	}

	return

}

// this will create a random avatar
func GenerateAvatar(uid uint) (err error) {

	if uid == 0 || uid == 1 {
		return errors.New("Invalid user id")
	}

	img := ImageType{
		avatar:     true,
		OrigWidth:  420,
		OrigHeight: 420,
		Ext:        ".png",
		image:      new(bytes.Buffer),
		Ib:         uid,
		MD5:        "fake",
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
	err = i.createThumbnail(128, 128)
	if err != nil {
		return
	}

	// copy the file to s3
	err = img.avatarToS3()
	if err != nil {
		return
	}

	return
}

// uploads to the avatar folder
func (i *ImageType) avatarToS3() (err error) {

	s3 := amazon.New()

	err = s3.Save(i.Thumbpath, fmt.Sprintf("avatars/%s", i.Thumbnail), i.mime, true)
	if err != nil {
		return
	}

	return
}
