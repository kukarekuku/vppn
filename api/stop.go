package api

import (
	"../profile"
	"../shared/command"
	"github.com/gin-gonic/gin"
)

func stopPost(c *gin.Context) {
	prfls := profile.GetProfiles()
	for _, prfl := range prfls {
		prfl.Stop()
	}

	command.CheckAndCleanWatch()

	c.JSON(200, nil)
}
