package main

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/gin-gonic/gin"
)

type Config struct {
	StereoPi StereoPiConfig `toml:"stereo_pi"`
	Server   ServerConfig   `toml:"server"`
}

type StereoPiConfig struct {
	IP   string `toml:"ip"`
	Port int    `toml:"port"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

var (
	leftCamera  Camera
	rightCamera Camera
)

func main() {
	var config Config

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Fatalf(
			"Failed to load configuration: %v",
			err,
		)
	}

	go connectStereoCamera(
		config.StereoPi.IP,
		config.StereoPi.Port,
		&leftCamera,
		&rightCamera,
	)

	router := gin.Default()

	router.GET("/", indexHandler)
	router.GET("/stream", streamHandler)

	address := fmt.Sprintf(
		"%s:%d",
		config.Server.Host,
		config.Server.Port,
	)

	log.Printf(
		"Web interface available at http://localhost:%d",
		config.Server.Port,
	)

	if err := router.Run(address); err != nil {
		log.Fatal(err)
	}
}
