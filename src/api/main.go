package main

import (
	"beruAPI/config"
	"beruAPI/server"
)

func main() {
	cfg := config.GetConfig()
	app := &server.App{}
	app.Init(cfg)
	app.Run(":"+cfg.App.AppPort)
}