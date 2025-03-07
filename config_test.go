package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Config(t *testing.T) {
	c, err := LoadConfig("./config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "dummy-key", c.ApiKey)
	assert.Equal(t, "dummy:token", c.TelegramToken)
	assert.Equal(t, _default_backup_dir, c.BackupDir)

}
