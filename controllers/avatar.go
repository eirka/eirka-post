package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/config"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	u "github.com/eirka/eirka-post/utils"
)

// AvatarController handles updating a users avatar
func AvatarController(c *gin.Context) {
	var err error
	req := c.Request

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	// check size of content
	if req.ContentLength > int64(config.Settings.Limits.ImageMaxSize) {
		c.JSON(http.StatusExpectationFailed, gin.H{"error_message": e.ErrImageSize.Error()})
		c.Error(e.ErrImageSize).SetMeta("AvatarController.ContentLength")
		return
	}

	// set max bytes reader
	req.Body = http.MaxBytesReader(c.Writer, req.Body, int64(config.Settings.Limits.ImageMaxSize))

	image := u.ImageType{}

	// Check if theres a file
	image.File, image.Header, err = req.FormFile("file")
	if err == http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrNoImage.Error()})
		c.Error(e.ErrNoImage).SetMeta("ThreadController.FormFile")
		return
	}

	// set the user id as the ib ;D
	image.Ib = userdata.Id

	// Save the image to a file
	err = image.SaveAvatar()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("AvatarController.SaveAvatar")
		return
	}

	c.Redirect(303, req.Referer())

	return

}
