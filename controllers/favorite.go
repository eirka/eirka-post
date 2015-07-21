package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	e "github.com/techjanitor/pram-post/errors"
	"github.com/techjanitor/pram-post/models"
	u "github.com/techjanitor/pram-post/utils"
)

// Add a favorite
type favoriteForm struct {
	Image uint `json:"image" binding:"required"`
}

// FavoriteController handles the creation of new threads
func FavoriteController(c *gin.Context) {
	var err error
	var ff favoriteForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(u.User)

	err = c.Bind(&ff)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Set parameters to FavoriteModel
	m := models.FavoriteModel{
		Uid:   userdata.Id,
		Ip:    c.ClientIP(),
		Image: ff.Image,
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err)
		return
	}

	// Check fav, if its there delete it because i dont want this to be too complicated
	err = m.Status()
	if err == e.ErrFavoriteRemoved {
		c.JSON(http.StatusOK, gin.H{"success_message": e.ErrFavoriteRemoved})
		return
	}
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	// Post data
	err = m.Post()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": "Favorite added"})

	return

}
