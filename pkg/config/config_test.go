package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestLoadConfigWithEnv(t *testing.T) {
	setProjectDir()

	expected := "test"
	t.Setenv("GICS_FHIR_BASE", expected)

	config, _ := parseConfig(".")

	assert.Equal(t, expected, config.Gics.Fhir.Base)
}

func TestLoadConfigFileNotFound(t *testing.T) {
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
