package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/eirka/eirka-libs/audit"
	"github.com/eirka/eirka-libs/auth"
	e "github.com/eirka/eirka-libs/errors"

	"github.com/eirka/eirka-post/models"
)

// Add a favorites
type favoritesForm struct {
	Image uint `json:"image" binding:"required"`
}

// FavoritessController handles the creation of new threads
func FavoritesController(c *gin.Context) {
	var err error
	var ff favoritesForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(auth.User)

	err = c.Bind(&ff)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err)
		return
	}

	// Set parameters to FavoritesModel
	m := models.FavoritesModel{
		Uid:   userdata.Id,
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
		c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditFavoriteRemoved})
		return
	} else if err != nil {
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

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditFavoriteAdded})

	return

}
