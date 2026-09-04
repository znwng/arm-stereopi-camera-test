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
		// -------------------------------------------------------------
		// Copy the current stereo state while holding the mutex.
		//
		// Since the receiver updates both cameras under the same
		// mutex, these values represent one complete stereo pair.
		// -------------------------------------------------------------

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

		currentLeftPairID := leftCamera.PairID
		currentRightPairID := rightCamera.PairID

		currentLeftTimestamp := leftCamera.Timestamp
		currentRightTimestamp := rightCamera.Timestamp

		currentLeftBytes := leftCamera.Bytes
		currentRightBytes := rightCamera.Bytes

		currentLeftFPS := leftCamera.FPS
		currentRightFPS := rightCamera.FPS

		mutex.RUnlock()

		// -------------------------------------------------------------
		// Send left frame
		//
		// Format:
		//
		// byte  0      = camera ID (0 = left)
		// bytes 1-8    = pair ID
		// bytes 9-16   = timestamp
		// bytes 17-N   = JPEG
		// -------------------------------------------------------------

		if currentLeftFrame != nil {
			data := make(
				[]byte,
				len(currentLeftFrame)+17,
			)

			data[0] = 0

			binary.BigEndian.PutUint64(
				data[1:9],
				currentLeftPairID,
			)

			binary.BigEndian.PutUint64(
				data[9:17],
				currentLeftTimestamp,
			)

			copy(
				data[17:],
				currentLeftFrame,
			)

			if err := conn.WriteMessage(
				websocket.BinaryMessage,
				data,
			); err != nil {
				return
			}
		}

		// -------------------------------------------------------------
		// Send right frame
		//
		// Format:
		//
		// byte  0      = camera ID (1 = right)
		// bytes 1-8    = pair ID
		// bytes 9-16   = timestamp
		// bytes 17-N   = JPEG
		// -------------------------------------------------------------

		if currentRightFrame != nil {
			data := make(
				[]byte,
				len(currentRightFrame)+17,
			)

			data[0] = 1

			binary.BigEndian.PutUint64(
				data[1:9],
				currentRightPairID,
			)

			binary.BigEndian.PutUint64(
				data[9:17],
				currentRightTimestamp,
			)

			copy(
				data[17:],
				currentRightFrame,
			)

			if err := conn.WriteMessage(
				websocket.BinaryMessage,
				data,
			); err != nil {
				return
			}
		}

		// -------------------------------------------------------------
		// Send statistics once per second
		// -------------------------------------------------------------

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
