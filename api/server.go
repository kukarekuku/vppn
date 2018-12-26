package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func Run() {

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	Register(router)

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
		err := server.ListenAndServe()
		if err != nil {
			log.Panic("main: Server error", err)
		}
	}()
}
