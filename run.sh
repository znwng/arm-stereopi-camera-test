#!/usr/bin/env bash

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	echo "Usage: $0 {server|client}"
	exit 1
}

case "$1" in
server)
	if ! command -v python3 &>/dev/null; then
		echo "Python 3 is not installed"
		exit 1
	fi

	echo "Python is installed: $(python3 --version)"

	cd "$ROOT_DIR/server"

	if [ ! -d ".venv" ]; then
		echo "Creating virtual environment..."
		python3 -m venv .venv
	fi

	echo "Installing server dependencies..."
	.venv/bin/pip install -r requirements.txt

	echo "Starting camera server..."
	exec .venv/bin/python main.py
	;;

client)
	if ! command -v go &>/dev/null; then
		echo "Go is not installed"
		exit 1
	fi

	echo "Go is installed: $(go version)"

	echo "Starting client..."
	cd "$ROOT_DIR/client"
	exec go run .
	;;

*)
	usage
	;;
esac
