// Api handlers.
package api

import (
	"crypto/subtle"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"net/http"
)

var (
	log = logging.MustGetLogger("api")
)

func Init(authKey string) {
	runServer(authKey)
}

// Recover panics
func Recovery(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Error("handlers: Handler panic", err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
		}
	}()

	c.Next()
}

// Log errors
func Errors(c *gin.Context) {
	c.Next()
	for _, err := range c.Errors {
		log.Error("handlers: Handler error", err)
	}
}

// Auth requests
func Auth(c *gin.Context) {
	if c.Request.Header.Get("Origin") != "" ||
		c.Request.Header.Get("Referer") != "" ||
		c.Request.Header.Get("User-Agent") != "pritunl" ||
		subtle.ConstantTimeCompare(
			[]byte(c.Request.Header.Get("Auth-Key")),
			[]byte(Key)) != 1 {

		c.AbortWithStatus(401)
		return
	}
	c.Next()
}

func Register(engine *gin.Engine) {
	// engine.Use(Auth)
	engine.Use(Recovery)
	engine.Use(Errors)

	engine.GET("/events", eventsGet)
	engine.GET("/profile", profileGet)
	engine.POST("/profile", profilePost)
	engine.DELETE("/profile", profileDel)
	engine.PUT("/token", tokenPut)
	engine.DELETE("/token", tokenDelete)
	engine.GET("/ping", pingGet)
	engine.POST("/stop", stopPost)
	engine.POST("/restart", restartPost)
	engine.GET("/status", statusGet)
	engine.POST("/wakeup", wakeupPost)
}
