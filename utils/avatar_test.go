package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAvatar(t *testing.T) {

	err := GenerateAvatar(0)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(1)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(2)
	assert.NoError(t, err, "An error was not expected")

}
