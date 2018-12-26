package main

import (
	"./api"
	"./auth"
	"./autoclean"
	"./config"
	"./shared/utils"
	"flag"
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("main")
)

func main() {
	// получаем параметры
	devPtr := flag.Bool("dev", false, "development mode")
	// если передали dev, то инитим приложение в dev mode
	flag.Parse()
	if *devPtr {
		config.Development = true
	}

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
