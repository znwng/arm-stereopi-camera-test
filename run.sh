#!/usr/bin/env bash

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	echo "Usage: $0 {server|client}"
	exit 1
}

case "$1" in
server)
	echo "Starting camera server..."
	cd "$ROOT_DIR/server"
	exec python3 camera_server.py
	;;

client)
	echo "Starting client..."
	cd "$ROOT_DIR/client"
	exec go run .
	;;

*)
	usage
	;;
esac
