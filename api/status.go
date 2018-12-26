package api

import (
	"../profile"
	"github.com/gin-gonic/gin"
)

type statusData struct {
	Status bool `json:"status"`
}

func statusGet(c *gin.Context) {
	data := &statusData{
		Status: profile.GetStatus(),
	}

	c.JSON(200, data)
}
