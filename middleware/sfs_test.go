package middleware

import (
	"errors"
	"testing"

	e "github.com/eirka/eirka-libs/errors"
	"github.com/stretchr/testify/assert"
)

func TestCheckStopForumSpam(t *testing.T) {

	err := CheckStopForumSpam("")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, errors.New("no ip provided"), err, "Error should match")
	}

	err = CheckStopForumSpam("127.0.0.1")
	assert.NoError(t, err, "An error was not expected")

	err = CheckStopForumSpam("188.143.232.34")
	if assert.Error(t, err, "An error was expected") {
		assert.Equal(t, e.ErrBlacklist, err, "Error should match")
	}

}
