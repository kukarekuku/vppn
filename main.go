package main

import (
	"./api"
	"./auth"
	"./autoclean"
	"./shared/utils"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("main")
)

func main() {
	// при старте первым аргументом передаем тип окружения "dev", "prod" etc
	// config.Local.Name = "dev"

	// инитим процесс
	err := utils.PidInit()
	if err != nil {
		log.Error("main: Panic", err)
		return
	}

	// инитим логи
	log.Info("main: Service starting")

	auth.Init()
	autoclean.Init()
	api.Init(auth.Key)
}
