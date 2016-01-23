package utils

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"

	_ "github.com/eirka/eirka-libs/config"
)

func TestCheckStopForumSpam(t *testing.T) {

	err := CheckStopForumSpam("")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, err, errors.New("no ip provided"), "Error should match")
	}

}
