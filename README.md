# Test Code for StereoPi

This repository contains the test code for the **StereoPi camera system**. It consists of two components:

* `server` — Runs on the StereoPi Compute Module and handles the camera streams.
* `client` — Runs on the host machine and provides the web interface.

The repository includes a `run.sh` script at its root for starting the appropriate component.

## Setup

Clone this repository on **both**:

1. The StereoPi Compute Module
2. The host machine (PC/laptop)

### On the StereoPi

Run:

```bash
./run.sh server
```

The script will:

1. Check that Python 3 is installed.
2. Create a Python virtual environment in `server/.venv/` if one does not already exist.
3. Install the dependencies from `server/requirements.txt`.
4. Start the camera server using `server/main.py`.

You do not need to manually create the virtual environment or install the Python dependencies.

### On the Host Machine

Run:

```bash
./run.sh client
```

The script will:

1. Check that Go is installed.
2. Start the Go client.

Once the client is running, open the address shown in the terminal in a web browser.

By default, it should be:

```text
http://localhost:8000
```

## Requirements

Both machines require a **working LAN connection** so that the server and client can communicate.

### StereoPi Compute Module

* Proper LAN connection to the host machine
* Python 3
* `python3-venv` package

The Python dependencies are listed in:

```text
server/requirements.txt
```

The `run.sh` script automatically creates and manages the virtual environment.

### Host Machine

* Proper LAN connection to the StereoPi
* **Go 1.27.0**

## Repository Structure

```text
.
├── client/
│   └── ...
├── server/
│   ├── main.py
│   ├── requirements.txt
│   └── .venv/
├── run.sh
└── README.md
```

> `server/.venv/` is created automatically by `run.sh` and should not be committed to Git.

## Architecture

The system consists of two components communicating over the LAN:

```text
┌─────────────────────────┐
│       StereoPi          │
│                         │
│  ┌───────────────────┐  │
│  │  Camera Server    │  │
│  │      Python       │  │
│  └─────────┬─────────┘  │
└────────────┼────────────┘
             │
             │ LAN
             │
┌────────────▼────────────┐
│      Host Machine       │
│                         │
│  ┌───────────────────┐  │
│  │      Client       │  │
│  │        Go         │  │
│  └─────────┬─────────┘  │
│            │            │
│            ▼            │
│       Web Browser       │
└─────────────────────────┘
```

The **server runs on the StereoPi**, while the **client runs on the host machine**. The two components communicate over the LAN.

