package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eirka/eirka-libs/audit"
	e "github.com/eirka/eirka-libs/errors"
	"github.com/eirka/eirka-libs/user"

	"github.com/eirka/eirka-post/models"
)

// Input from change email form
type emailForm struct {
	Ib    uint   `json:"ib" binding:"required"`
	Email string `json:"email" binding:"required"`
}

// EmailController handles updating a users email
func EmailController(c *gin.Context) {
	var err error
	var ef emailForm

	// get userdata from session middleware
	userdata := c.MustGet("userdata").(user.User)

	err = c.Bind(&ef)
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInvalidParam))
		c.Error(err).SetMeta("EmailController.Bind")
		return
	}

	// Set parameters to EmailModel
	m := models.EmailModel{
		UID:   userdata.ID,
		Email: ef.Email,
	}

	// Validate input
	err = m.Validate()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error_message": err.Error()})
		c.Error(err).SetMeta("EmailController.Validate")
		return
	}

	// update password
	err = m.Update()
	if err != nil {
		c.JSON(e.ErrorMessage(e.ErrInternalError))
		c.Error(err).SetMeta("EmailController.Update")
		return
	}

	c.JSON(http.StatusOK, gin.H{"success_message": audit.AuditEmailUpdate})

	audit := audit.Audit{
		User:   userdata.ID,
		Ib:     ef.Ib,
		Type:   audit.UserLog,
		IP:     c.ClientIP(),
		Action: audit.AuditEmailUpdate,
		Info:   m.Name,
	}

	// submit audit
	err = audit.Submit()
	if err != nil {
		c.Error(err).SetMeta("EmailController.audit.Submit")
	}

	return

}
