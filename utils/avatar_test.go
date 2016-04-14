package utils

import (
	"testing"

	"github.com/eirka/eirka-libs/config"
	"github.com/eirka/eirka-libs/db"
	"github.com/stretchr/testify/assert"

	local "github.com/eirka/eirka-post/config"
)

func TestGenerateAvatar(t *testing.T) {

	// Database connection settings
	dbase := db.Database{
		User:           local.Settings.Database.User,
		Password:       local.Settings.Database.Password,
		Proto:          local.Settings.Database.Proto,
		Host:           local.Settings.Database.Host,
		Database:       local.Settings.Database.Database,
		MaxIdle:        local.Settings.Database.MaxIdle,
		MaxConnections: local.Settings.Database.MaxConnections,
	}

	// Set up DB connection
	dbase.NewDb()

	// Get limits and stuff from database
	config.GetDatabaseSettings()

	err := GenerateAvatar(0)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(1)
	assert.Error(t, err, "An error was expected")

	err = GenerateAvatar(2)
	assert.NoError(t, err, "An error was not expected")

}
