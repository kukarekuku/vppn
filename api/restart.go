package api

import (
	"../profile"
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func restartPost(c *gin.Context) {
	logrus.Warn("handlers: Restarting...")

	profile.RestartProfiles(false)

	c.JSON(200, nil)
}
