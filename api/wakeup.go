package api

import (
	"../shared/events"
	"../shared/utils"
	"github.com/gin-gonic/gin"
	"time"
)

func wakeupPost(c *gin.Context) {
	evt := &events.Event{
		Id:   utils.Uuid(),
		Type: "wakeup",
	}
	evt.Init()

	for i := 0; i < 50; i++ {
		time.Sleep(5 * time.Millisecond)
		if time.Since(events.LastAwake) < 200*time.Millisecond {
			c.String(200, "")
			return
		}
	}

	c.String(404, "")
}
