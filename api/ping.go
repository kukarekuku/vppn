package api

import (
	"github.com/gin-gonic/gin"
)

func pingGet(c *gin.Context) {
	c.String(200, "")
}
