package main

import (
	"consented/pkg/config"
	"consented/pkg/web"
	"github.com/rs/zerolog/log"
)

func main() {
	appConfig := config.LoadConfig()

	server := web.NewServer(appConfig)
	log.Fatal().Err(server.Run()).Msg("Server failed to run")
}
