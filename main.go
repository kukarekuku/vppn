package main

import (
	"./api"
	"./auth"
	"./config"
	"./profile"
	"./shared/command"
	"./shared/utils"
	"./watch"
	"context"
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	err = auth.Init()
	if err != nil {
		log.Error("main: Failed to init auth", err)
		return
	}

	err = command.CheckAndClean()
	if err != nil {
		log.Error("main: Failed to run check and clean", err)
		return
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	api.Register(router)

	watch.StartWatch()

	server := &http.Server{
		Addr:           "127.0.0.1:9780",
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 4096,
	}

	go func() {
		defer func() {
			recover()
		}()
		err = server.ListenAndServe()
		if err != nil {
			log.Error("main: Server error", err)
			return
		}
	}()

	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	webCtx, webCancel := context.WithTimeout(
		context.Background(),
		1*time.Second,
	)
	defer webCancel()

	func() {
		defer func() {
			recover()
		}()
		server.Shutdown(webCtx)
		server.Close()
	}()

	time.Sleep(250 * time.Millisecond)

	prfls := profile.GetProfiles()
	for _, prfl := range prfls {
		prfl.Stop()
	}

	time.Sleep(750 * time.Millisecond)
}
