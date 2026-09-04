import socket
import struct
import threading
import time
import tomllib

import cv2
from picamera2 import Picamera2

with open("config.toml", "rb") as file:
    config = tomllib.load(file)


HOST = config["host"]

WIDTH = config["width"]
HEIGHT = config["height"]

FPS = config["fps"]
CAMERA_FPS = config["camera_fps"]

JPEG_QUALITY = config["jpeg_quality"]

STEREO_PORT = config["stereo_port"]


left_camera = Picamera2(0)
right_camera = Picamera2(1)


left_camera.configure(
    left_camera.create_preview_configuration(
        main={
            "size": (WIDTH, HEIGHT),
            "format": "RGB888",
        },
        controls={
            "FrameRate": float(CAMERA_FPS),
        },
    )
)

right_camera.configure(
    right_camera.create_preview_configuration(
        main={
            "size": (WIDTH, HEIGHT),
            "format": "RGB888",
        },
        controls={
            "FrameRate": float(CAMERA_FPS),
        },
    )
)


left_camera.start()
right_camera.start()


def create_server(port):
    server = socket.socket(
        socket.AF_INET,
        socket.SOCK_STREAM,
    )

    server.setsockopt(
        socket.SOL_SOCKET,
        socket.SO_REUSEADDR,
        1,
    )

    server.bind((HOST, port))

    server.listen(1)

    return server


stereo_server = create_server(STEREO_PORT)


print("Configuration:")
print(f"  Resolution: {WIDTH}x{HEIGHT}")
print(f"  Target FPS: {FPS}")
print(f"  Camera FPS: {CAMERA_FPS}")
print(f"  JPEG quality: {JPEG_QUALITY}")
print()

print("Server started:")
print(f"  stereo_camera -> port {STEREO_PORT}")


ACK = b"received both frames"


def recv_exact(conn, size):
    data = bytearray()

    while len(data) < size:
        chunk = conn.recv(size - len(data))

        if not chunk:
            raise ConnectionError("Client disconnected while waiting for ACK")

        data.extend(chunk)

    return bytes(data)


def capture_camera(
    camera,
    result,
    key,
    barrier,
):
    barrier.wait()

    frame = camera.capture_array()

    timestamp = time.monotonic_ns()

    result[key] = (
        frame,
        timestamp,
    )


def encode_frame(frame):
    gray = cv2.cvtColor(
        frame,
        cv2.COLOR_RGB2GRAY,
    )

    success, encoded = cv2.imencode(
        ".jpg",
        gray,
        [
            cv2.IMWRITE_JPEG_QUALITY,
            JPEG_QUALITY,
        ],
    )

    if not success:
        raise RuntimeError("Failed to encode frame")

    return encoded.tobytes()


def stream_stereo(
    server,
):
    frame_time = 1.0 / FPS

    # Unique ID for each left/right stereo pair.
    pair_id = 0

    while True:

        print("stereo_camera: waiting for laptop...")

        conn = None

        try:

            conn, addr = server.accept()

            print(f"stereo_camera: connected from {addr}")

            next_frame_time = time.monotonic()

            while True:

                wait_time = next_frame_time - time.monotonic()

                if wait_time > 0:
                    time.sleep(wait_time)

                # ---------------------------------------------------------
                # Capture both cameras concurrently
                # ---------------------------------------------------------

                results = {}

                barrier = threading.Barrier(2)

                left_capture_thread = threading.Thread(
                    target=capture_camera,
                    args=(
                        left_camera,
                        results,
                        "left",
                        barrier,
                    ),
                )

                right_capture_thread = threading.Thread(
                    target=capture_camera,
                    args=(
                        right_camera,
                        results,
                        "right",
                        barrier,
                    ),
                )

                left_capture_thread.start()
                right_capture_thread.start()

                left_capture_thread.join()
                right_capture_thread.join()

                # ---------------------------------------------------------
                # Get captured frames
                # ---------------------------------------------------------

                left_frame, left_timestamp = results["left"]

                right_frame, right_timestamp = results["right"]

                # ---------------------------------------------------------
                # Increment pair ID
                #
                # One pair ID corresponds to exactly one left/right pair.
                # ---------------------------------------------------------

                pair_id += 1

                # ---------------------------------------------------------
                # JPEG compression
                # ---------------------------------------------------------

                left_data = encode_frame(left_frame)

                right_data = encode_frame(right_frame)

                # ---------------------------------------------------------
                # Calculate capture time difference
                # ---------------------------------------------------------

                timestamp_difference_us = abs(left_timestamp - right_timestamp) / 1000

                print(
                    f"Pair {pair_id}: "
                    f"L/R difference = "
                    f"{timestamp_difference_us:.1f} us"
                )

                # ---------------------------------------------------------
                # Send pair ID
                #
                # uint64:
                #   !Q = unsigned 64-bit integer, network byte order
                # ---------------------------------------------------------

                conn.sendall(
                    struct.pack(
                        "!Q",
                        pair_id,
                    )
                )

                # ---------------------------------------------------------
                # Send left frame
                # ---------------------------------------------------------

                conn.sendall(
                    struct.pack(
                        "!I",
                        len(left_data),
                    )
                )

                conn.sendall(
                    struct.pack(
                        "!Q",
                        left_timestamp,
                    )
                )

                conn.sendall(left_data)

                # ---------------------------------------------------------
                # Send right frame
                # ---------------------------------------------------------

                conn.sendall(
                    struct.pack(
                        "!I",
                        len(right_data),
                    )
                )

                conn.sendall(
                    struct.pack(
                        "!Q",
                        right_timestamp,
                    )
                )

                conn.sendall(right_data)

                # ---------------------------------------------------------
                # Wait for client confirmation
                # ---------------------------------------------------------

                response = recv_exact(
                    conn,
                    len(ACK),
                )

                if response != ACK:
                    raise ConnectionError(f"Unexpected client response: {response!r}")

                # ---------------------------------------------------------
                # Schedule the next pair
                # ---------------------------------------------------------

                next_frame_time += frame_time

        except (
            BrokenPipeError,
            ConnectionResetError,
            ConnectionAbortedError,
            ConnectionError,
        ) as error:

            print(f"stereo_camera: laptop disconnected: {error}")

        except OSError as error:

            print(f"stereo_camera: connection error: {error}")

        finally:

            if conn is not None:

                try:
                    conn.close()
                except OSError:
                    pass


stereo_thread = threading.Thread(
    target=stream_stereo,
    args=(stereo_server,),
    daemon=True,
)

stereo_thread.start()


try:

    while True:
        time.sleep(1)

except KeyboardInterrupt:

    print()
    print("Shutting down...")


finally:

    stereo_server.close()

    left_camera.stop()
    right_camera.stop()
