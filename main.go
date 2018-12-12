package main

import (
	"./auth"
	"./autoclean"
	"./constants"
	"./handlers"
	"./logger"
	"./profile"
	"./utils"
	"./watch"
	"context"
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	devPtr := flag.Bool("dev", false, "development mode")
	flag.Parse()
	if *devPtr {
		constants.Development = true
	}

	err := utils.PidInit()
	if err != nil {
		panic(err)
	}

	logger.Init()

	logrus.WithFields(logrus.Fields{
		"version": constants.Version,
	}).Info("main: Service starting")

	defer func() {
		panc := recover()
		if panc != nil {
			logrus.WithFields(logrus.Fields{
				"stack": string(debug.Stack()),
				"panic": panc,
			}).Error("main: Panic")
			panic(panc)
		}
	}()

	err = auth.Init()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("main: Failed to init auth")
		panic(err)
	}

	err = autoclean.CheckAndClean()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("main: Failed to run check and clean")
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	handlers.Register(router)

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
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("main: Server error")
			panic(err)
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
