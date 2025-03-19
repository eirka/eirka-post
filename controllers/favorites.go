package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/audit"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
)

// favoritesForm contains the user input for the favorites toggle
type favoritesForm struct {
	Image uint `json:"image" binding:"required"`
}

// FavoritesController handles adding or removing an image from a user's favorites
// Acts as a toggle: adds if not favorited, removes if already favorited
func FavoritesController(c *gin.Context) {
	var err error
	var ff favoritesForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&ff)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("FavoritesController.Bind")
		return
	}

	// Set parameters to FavoritesModel
	m := models.FavoritesModel{
		UID:   userdata.ID,
		Image: ff.Image,
	}

	// Validate input parameters
	err = m.ValidateInput()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("FavoritesController.ValidateInput")
		return
	}

	// Check if image is already favorited and handle accordingly
	err = m.Status()
	if err == e.ErrFavoriteRemoved {
		// Favorite was removed successfully
		c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditFavoriteRemoved})
		return
	} else if err == e.ErrNotFound {
		// Image doesn't exist
		c.JSON(e.ErrorMessage(e.ErrNotFound))
		c.Error(err).SetMeta("FavoritesController.Status")
		return
	} else if err != nil {
		// Other database error
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("FavoritesController.Status")
		return
	}

	// Post data (add favorite)
	err = m.Post()
	if err == e.ErrNotFound {
		// Image doesn't exist
		c.JSON(e.ErrorMessage(e.ErrNotFound))
		c.Error(err).SetMeta("FavoritesController.Post")
		return
	} else if err != nil {
		// Other database error
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("FavoritesController.Post")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditFavoriteAdded})
}
