package main

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Stats struct {
	FPS      float64 `json:"fps"`
	Transfer float64 `json:"transfer"`
}

type CameraStats struct {
	Left  Stats `json:"left"`
	Right Stats `json:"right"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func streamHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(
		c.Writer,
		c.Request,
		nil,
	)

	if err != nil {
		return
	}

	defer conn.Close()

	lastStatsTime := time.Now()

	var lastLeftBytes uint64
	var lastRightBytes uint64

	for {
		mutex.RLock()

		var currentLeftFrame []byte

		if leftCamera.Frame != nil {
			currentLeftFrame = append(
				[]byte(nil),
				leftCamera.Frame...,
			)
		}

		var currentRightFrame []byte

		if rightCamera.Frame != nil {
			currentRightFrame = append(
				[]byte(nil),
				rightCamera.Frame...,
			)
		}

		currentLeftTimestamp := leftCamera.Timestamp
		currentRightTimestamp := rightCamera.Timestamp

		currentLeftBytes := leftCamera.Bytes
		currentRightBytes := rightCamera.Bytes

		currentLeftFPS := leftCamera.FPS
		currentRightFPS := rightCamera.FPS

		mutex.RUnlock()

		if currentLeftFrame != nil {
			data := make(
				[]byte,
				len(currentLeftFrame)+9,
			)

			data[0] = 0

			binary.BigEndian.PutUint64(
				data[1:9],
				currentLeftTimestamp,
			)

			copy(
				data[9:],
				currentLeftFrame,
			)

			if err := conn.WriteMessage(
				websocket.BinaryMessage,
				data,
			); err != nil {
				return
			}
		}

		if currentRightFrame != nil {
			data := make(
				[]byte,
				len(currentRightFrame)+9,
			)

			data[0] = 1

			binary.BigEndian.PutUint64(
				data[1:9],
				currentRightTimestamp,
			)

			copy(
				data[9:],
				currentRightFrame,
			)

			if err := conn.WriteMessage(
				websocket.BinaryMessage,
				data,
			); err != nil {
				return
			}
		}

		now := time.Now()

		elapsed := now.Sub(
			lastStatsTime,
		).Seconds()

		if elapsed >= 1.0 {
			leftRate :=
				float64(
					currentLeftBytes-lastLeftBytes,
				) *
					8 /
					elapsed /
					1_000_000

			rightRate :=
				float64(
					currentRightBytes-lastRightBytes,
				) *
					8 /
					elapsed /
					1_000_000

			stats := CameraStats{
				Left: Stats{
					FPS:      currentLeftFPS,
					Transfer: leftRate,
				},
				Right: Stats{
					FPS:      currentRightFPS,
					Transfer: rightRate,
				},
			}

			jsonData, err := json.Marshal(stats)

			if err != nil {
				return
			}

			if err := conn.WriteMessage(
				websocket.TextMessage,
				jsonData,
			); err != nil {
				return
			}

			lastLeftBytes = currentLeftBytes
			lastRightBytes = currentRightBytes
			lastStatsTime = now
		}

		time.Sleep(10 * time.Millisecond)
	}
}
