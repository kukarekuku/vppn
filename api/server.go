package api

import (
	"../profile"
	"../watch"
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	Key = ""
)

func Init(authKey string) {
	Key = authKey

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	Register(router)

	watch.StartWatch()

	server := &http.Server{
		Addr:           "127.0.0.1:9780",
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 4096,
	}

	run(server)

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

	getProfiles()
}

func run(server *http.Server) {
	defer func() {
		recover()
	}()

	err := server.ListenAndServe()
	if err != nil {
		log.Error("main: Server error", err)
		return
	}
}

func getProfiles() {
	time.Sleep(250 * time.Millisecond)

	prfls := profile.GetProfiles()
	for _, prfl := range prfls {
		prfl.Stop()
	}

	time.Sleep(750 * time.Millisecond)
}
