package main

import (
	"consented/pkg/config"
	"consented/pkg/web"
)

func main() {
	appConfig := config.LoadConfig()

	server := web.NewServer(appConfig)
	server.Run()
}
