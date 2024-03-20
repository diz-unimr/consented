package config

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestLoadConfigConfiguresLogger(t *testing.T) {
	setProjectDir()

	expected := "debug"
	t.Setenv("APP_LOG_LEVEL", expected)

	LoadConfig()

	assert.Equal(t, zerolog.LevelDebugValue, zerolog.GlobalLevel().String())
}

func TestParseConfigWithEnv(t *testing.T) {
	setProjectDir()

	expected := "test"
	t.Setenv("GICS_FHIR_BASE", expected)

	config := LoadConfig()

	assert.Equal(t, expected, config.Gics.Fhir.Base)
}

func TestParseConfigFileNotFound(t *testing.T) {
	setProjectDir()

	// config file not found
	_, err := parseConfig("./bla")

	assert.ErrorIs(t, err, err.(viper.ConfigFileNotFoundError))
}

func setProjectDir() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	_ = os.Chdir(dir)

	viper.Reset()
}
