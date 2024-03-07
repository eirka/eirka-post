package middleware

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestInTimeSpan(t *testing.T) {
	assert := assert.New(t)

	testTrue := []struct {
		start string
		end   string
		check string
	}{
		{"08:00:00", "15:00:00", "10:00:00"},
		{"08:00:00", "15:00:00", "11:30:00"},
		{"08:00:00", "15:00:00", "15:00:00"},
		{"08:00:00", "15:00:00", "08:00:00"},
	}

	for _, t := range testTrue {

		start, _ := time.Parse(time.TimeOnly, t.start)
		end, _ := time.Parse(time.TimeOnly, t.end)
		check, _ := time.Parse(time.TimeOnly, t.check)

		assert.True(inTimeSpan(start, end, check), "should be true")
	}

	testFalse := []struct {
		start string
		end   string
		check string
	}{
		{"08:00:00", "15:00:00", "20:00:00"},
		{"08:00:00", "15:00:00", "06:30:00"},
		{"08:00:00", "15:00:00", "15:00:01"},
	}

	for _, t := range testFalse {

		start, _ := time.Parse(time.TimeOnly, t.start)
		end, _ := time.Parse(time.TimeOnly, t.end)
		check, _ := time.Parse(time.TimeOnly, t.check)

		assert.False(inTimeSpan(start, end, check), "should be false")
	}
}

func TestGoodnight(t *testing.T) {

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// use middleware
	router.Use(Goodnight())

	router.POST("/reply", func(c *gin.Context) {
		c.String(200, "OK")
	})

	testTimeInRange, _ := time.Parse(time.TimeOnly, "10:00:00")

	timeNow = testTimeInRange.UTC

	first := performRequest(router, "POST", "/reply")
	assert.Equal(t, 403, first.Code, "HTTP request code should match")

	testTimeOutRange, _ := time.Parse(time.TimeOnly, "06:00:00")

	timeNow = testTimeOutRange.UTC

	second := performRequest(router, "POST", "/reply")
	assert.Equal(t, 200, second.Code, "HTTP request code should match")

}
