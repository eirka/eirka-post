package utils

import (
	"errors"
	"fmt"
	"github.com/1l0/identicon"
	"path/filepath"

	"github.com/eirka/eirka-libs/amazon"

	local "github.com/eirka/eirka-post/config"
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
	err = i.createThumbnail(200, 200)
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

	img := ImageType{
		avatar:     true,
		OrigWidth:  420,
		OrigHeight: 420,
		Ext:        ".png",
	}

	img.makeFilenames()

	img.Thumbnail = fmt.Sprintf("%d.png", uid)
	img.Thumbpath = filepath.Join(local.Settings.Directories.ThumbnailDir, img.Thumbnail)

	id := identicon.New()

	err = id.GeneratePNGToFile(img.Filepath)
	if err != nil {
		return
	}

	// create a thumbnail
	err = img.createThumbnail(200, 200)
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

	err = s3.Save(i.Thumbpath, fmt.Sprintf("avatars/%s", i.Thumbnail), "image/png", true)
	if err != nil {
		return
	}

	return
}
