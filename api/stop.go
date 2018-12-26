package api

import (
	"../autoclean"
	"../profile"
	"github.com/gin-gonic/gin"
)

func stopPost(c *gin.Context) {
	prfls := profile.GetProfiles()
	for _, prfl := range prfls {
		prfl.Stop()
	}

	autoclean.CheckAndCleanWatch()

	c.JSON(200, nil)
}
