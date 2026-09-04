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
	PairID     uint64
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

func receivePairID(
	conn net.Conn,
) (uint64, error) {
	data, err := recvExact(
		conn,
		8,
	)

	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(data), nil
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

	frameSize := binary.BigEndian.Uint32(header)

	timestampData, err := recvExact(
		conn,
		8,
	)

	if err != nil {
		return nil, 0, err
	}

	timestamp := binary.BigEndian.Uint64(timestampData)

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
	pairID uint64,
	now time.Time,
) {
	camera.Frame = frame
	camera.Timestamp = timestamp
	camera.PairID = pairID
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
		// -------------------------------------------------------------
		// Receive pair ID
		//
		// One pair ID corresponds to exactly one left/right pair.
		// -------------------------------------------------------------

		pairID, err := receivePairID(conn)

		if err != nil {
			log.Printf(
				"StereoPi disconnected while receiving pair ID: %v",
				err,
			)

			return
		}

		// -------------------------------------------------------------
		// Receive left frame
		// -------------------------------------------------------------

		leftFrame, leftTimestamp, err := receiveFrame(conn)

		if err != nil {
			log.Printf(
				"StereoPi disconnected while receiving left frame for pair %d: %v",
				pairID,
				err,
			)

			return
		}

		// -------------------------------------------------------------
		// Receive right frame
		// -------------------------------------------------------------

		rightFrame, rightTimestamp, err := receiveFrame(conn)

		if err != nil {
			log.Printf(
				"StereoPi disconnected while receiving right frame for pair %d: %v",
				pairID,
				err,
			)

			return
		}

		// -------------------------------------------------------------
		// Calculate left/right timestamp difference
		// -------------------------------------------------------------

		var timestampDifference uint64

		if leftTimestamp > rightTimestamp {
			timestampDifference =
				leftTimestamp - rightTimestamp
		} else {
			timestampDifference =
				rightTimestamp - leftTimestamp
		}

		timestampDifferenceUS :=
			float64(timestampDifference) / 1000.0

		fmt.Printf(
			"\r\033[KReceived stereo pair %d: L/R difference = %.1f us",
			pairID,
			timestampDifferenceUS,
		)

		// -------------------------------------------------------------
		// Update both cameras atomically
		//
		// The mutex ensures that the HTTP/WebSocket side never sees
		// a partially updated stereo pair.
		// -------------------------------------------------------------

		now = time.Now()

		mutex.Lock()

		updateCamera(
			leftCamera,
			leftFrame,
			leftTimestamp,
			pairID,
			now,
		)

		updateCamera(
			rightCamera,
			rightFrame,
			rightTimestamp,
			pairID,
			now,
		)

		mutex.Unlock()

		// -------------------------------------------------------------
		// Tell StereoPi that the complete pair was received.
		// -------------------------------------------------------------

		_, err = conn.Write([]byte(ackMessage))

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
	ip string,
	port int,
	leftCamera *Camera,
	rightCamera *Camera,
) {
	address := net.JoinHostPort(
		ip,
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

		log.Printf("Connected to StereoPi!")

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
