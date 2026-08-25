package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Camera struct {
	Frame      []byte
	Timestamp  uint64
	Bytes      uint64
	FPS        float64
	FrameCount uint64
	LastStats  time.Time
}

var mutex sync.RWMutex

const ackMessage = "received both frames"

func recvExact(
	conn net.Conn,
	size int,
) ([]byte, error) {
	data := make([]byte, size)

	_, err := io.ReadFull(
		conn,
		data,
	)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func receiveFrame(
	conn net.Conn,
) ([]byte, uint64, error) {
	header, err := recvExact(
		conn,
		4,
	)

	if err != nil {
		return nil, 0, err
	}

	frameSize := binary.BigEndian.Uint32(
		header,
	)

	timestampData, err := recvExact(
		conn,
		8,
	)

	if err != nil {
		return nil, 0, err
	}

	timestamp := binary.BigEndian.Uint64(
		timestampData,
	)

	frame, err := recvExact(
		conn,
		int(frameSize),
	)

	if err != nil {
		return nil, 0, err
	}

	return frame, timestamp, nil
}

func updateCamera(
	camera *Camera,
	frame []byte,
	timestamp uint64,
	now time.Time,
) {
	camera.Frame = frame
	camera.Timestamp = timestamp
	camera.Bytes += uint64(len(frame))
	camera.FrameCount++

	elapsed := now.Sub(
		camera.LastStats,
	).Seconds()

	if elapsed >= 1.0 {
		camera.FPS =
			float64(camera.FrameCount) / elapsed

		camera.FrameCount = 0
		camera.LastStats = now
	}
}

func stereoReceiver(
	conn net.Conn,
	leftCamera *Camera,
	rightCamera *Camera,
) {
	defer conn.Close()

	now := time.Now()

	leftCamera.LastStats = now
	rightCamera.LastStats = now

	for {
		leftFrame, leftTimestamp, err := receiveFrame(
			conn,
		)

		if err != nil {
			log.Printf(
				"StereoPi disconnected while receiving left frame: %v",
				err,
			)

			return
		}

		rightFrame, rightTimestamp, err := receiveFrame(
			conn,
		)

		if err != nil {
			log.Printf(
				"StereoPi disconnected while receiving right frame: %v",
				err,
			)

			return
		}

		now = time.Now()

		mutex.Lock()

		updateCamera(
			leftCamera,
			leftFrame,
			leftTimestamp,
			now,
		)

		updateCamera(
			rightCamera,
			rightFrame,
			rightTimestamp,
			now,
		)

		mutex.Unlock()

		_, err = conn.Write(
			[]byte(ackMessage),
		)

		if err != nil {
			log.Printf(
				"Failed to send stereo ACK: %v",
				err,
			)

			return
		}
	}
}

func connectStereoCamera(
	port int,
	leftCamera *Camera,
	rightCamera *Camera,
) {
	address := net.JoinHostPort(
		stereoPiIP,
		fmt.Sprintf("%d", port),
	)

	for {
		log.Printf(
			"Connecting to StereoPi at %s...",
			address,
		)

		conn, err := net.Dial(
			"tcp",
			address,
		)

		if err != nil {
			log.Printf(
				"Failed to connect to StereoPi: %v",
				err,
			)

			time.Sleep(2 * time.Second)

			continue
		}

		log.Printf(
			"Connected to StereoPi!",
		)

		stereoReceiver(
			conn,
			leftCamera,
			rightCamera,
		)

		log.Printf(
			"StereoPi connection lost. Reconnecting...",
		)

		time.Sleep(2 * time.Second)
	}
}
