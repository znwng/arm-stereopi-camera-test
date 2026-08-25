package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

const (
	stereoPiIP = "172.16.48.240"

	stereoPort = 5000

	httpHost = "0.0.0.0"
	httpPort = 8000
)

var (
	leftCamera  Camera
	rightCamera Camera
)

func main() {
	go connectStereoCamera(
		stereoPort,
		&leftCamera,
		&rightCamera,
	)

	router := gin.Default()

	router.GET("/", indexHandler)
	router.GET("/stream", streamHandler)

	address := fmt.Sprintf(
		"%s:%d",
		httpHost,
		httpPort,
	)

	log.Printf(
		"Web interface available at http://localhost:%d",
		httpPort,
	)

	if err := router.Run(address); err != nil {
		log.Fatal(err)
	}
}
