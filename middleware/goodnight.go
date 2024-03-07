package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	e "github.com/eirka/eirka-libs/errors"
)

var (
	timeNow            = time.Now
	errPostingDisabled = "posting is temporarily disabled"
)

// Goodnight will disable posting during a time range
func Goodnight() gin.HandlerFunc {
	return func(c *gin.Context) {

		start, _ := time.Parse(time.TimeOnly, "08:00:00")
		end, _ := time.Parse(time.TimeOnly, "15:00:00")

		if inTimeSpan(start, end, timeNow()) {
			c.JSON(e.ErrorMessage(&e.RequestError{ErrorString: errPostingDisabled, ErrorCode: http.StatusBadRequest}))
			c.Error(errors.New(errPostingDisabled)).SetMeta("Goodnight")
			c.Abort()
			return
		}

		c.Next()

	}
}

// inTimeSpan checks if a time is within a time range
func inTimeSpan(start, end, check time.Time) bool {
	return !check.Before(start) && !check.After(end)
}
