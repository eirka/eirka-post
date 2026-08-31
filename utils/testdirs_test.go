package utils

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	local "github.com/eirka/eirka-post/config"
)

func init() {
	dirs := []string{
		local.Settings.Directories.ImageDir,
		local.Settings.Directories.ThumbnailDir,
		local.Settings.Directories.AvatarDir,
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("failed to mkdir %s: %v", dir, err))
		}
	}
}

func TestConfiguredImageDirsExist(t *testing.T) {
	dirs := []string{
		local.Settings.Directories.ImageDir,
		local.Settings.Directories.ThumbnailDir,
		local.Settings.Directories.AvatarDir,
	}
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		assert.NoError(t, err, "configured directory should exist: %s", dir)
		if err != nil {
			continue
		}
		assert.True(t, info.IsDir(), "configured path should be a directory: %s", dir)
	}
}
