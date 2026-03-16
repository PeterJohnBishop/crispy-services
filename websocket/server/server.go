package server

import (
	"fmt"
	"log"
	"os"
	"time"
	"websocket/server/ws"

	"github.com/gin-gonic/gin"
)

var hub *ws.Hub

func ServeGin() {
	log.Println("Ordering Websocket Gin...")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	r.GET("/", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"identity": "websocket-server",
		})
	})

	hub := ws.NewHub()
	go hub.Run()
	r.GET("/ws", func(ctx *gin.Context) {
		log.Printf("Connected to WebSocket")
		ws.HandleWebsocket(hub, ctx)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	config := fmt.Sprintf(":%s", port)
	log.Printf("Serving Websocket Gin on port :%s", port)
	go func() {
		r.Run(config)
	}()

	time.Sleep(1 * time.Second)
	ws.StartInternalClient("localhost:8081", "/ws")
	select {}
}
