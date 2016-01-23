package utils

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/eirka/eirka-libs/config"
)

func TestCheckStopForumSpam(t *testing.T) {

	err := CheckStopForumSpam("")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("no ip provided"), "Error should match")
	}

	config.Settings.StopForumSpam.Confidence = 20

	err = CheckStopForumSpam("127.0.0.1")
	assert.NoError(t, err, "An error was not expected")

}
