package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

	image := u.ImageType{}

	// Check if theres a file
	image.File, image.Header, err = req.FormFile("file")
	if err == http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": e.ErrNoImage.Error()})
		c.Error(e.ErrNoImage).SetMeta("ThreadController.FormFile")
		return
	}

	// set the user id as the ib ;D
	image.Ib = userdata.ID

	// Save the image to a file
	err = image.SaveAvatar()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("AvatarController.SaveAvatar")
		return
	}

	c.Redirect(303, req.Referer())

}
