package api

import (
	"../profile"
	"../shared/events"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
	pingWait     = 40 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		HandshakeTimeout: 30 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
	}
)

func eventsGet(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	conn.SetReadDeadline(time.Now().Add(pingWait))
	conn.SetPongHandler(func(x string) (err error) {
		conn.SetReadDeadline(time.Now().Add(pingWait))
		return
	})

	list := events.NewListener()

	ticker := time.NewTicker(pingInterval)

	defer func() {
		ticker.Stop()
		conn.Close()
		list.Close()
	}()

	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Panic("events: Panic", err)
			}
		}()

		for {
			_, msgByt, err := conn.NextReader()
			if err != nil {
				conn.Close()
				break
			}

			msg, err := ioutil.ReadAll(msgByt)
			if err != nil {
				continue
			}

			if string(msg) == "awake" {
				events.LastAwake = time.Now()
			}
		}
	}()

	for {
		select {
		case evt, ok := <-list.Listen():
			if !ok {
				conn.WriteControl(websocket.CloseMessage, []byte{},
					time.Now().Add(writeTimeout))
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err = conn.WriteJSON(evt)
			if err != nil {
				return
			}
		case <-ticker.C:
			err = conn.WriteControl(websocket.PingMessage, []byte{},
				time.Now().Add(writeTimeout))
			if err != nil {
				return
			}

			profile.Ping = time.Now()
		}
	}
}
