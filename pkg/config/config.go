package config

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"strings"
)

type AppConfig struct {
	App App `mapstructure:"app"`
}

type Http struct {
	Auth Auth   `mapstructure:"auth"`
	Port string `mapstructure:"port"`
}

type App struct {
	LogLevel string `mapstructure:"log-level"`
	Http     Http   `mapstructure:"http"`
}

type Auth struct {
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

func LoadConfig() AppConfig {
	c, err := parseConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to load config file")
		os.Exit(1)
	}

	// log level
	logLevel, err := zerolog.ParseLevel(c.App.LogLevel)
	if err != nil {
		zerolog.SetGlobalLevel(logLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// pretty logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	return *c
}

func parseConfig(path string) (config *AppConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("yml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`, `-`, `_`))

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	err = viper.Unmarshal(&config)
	return config, err
}
