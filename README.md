# Test Code for StereoPi

This repository contains the test code for the StereoPi camera system. It consists of two components:

* `server` — Runs on the StereoPi Compute Module and handles the camera streams.
* `client` — Runs on the host machine and provides the web interface.

## Setup

Clone this repository on **both**:

1. The StereoPi Compute Module
2. The host machine (PC/laptop)

The repository includes a `run.sh` script at its root for starting the appropriate component.

### On the StereoPi

Run:

```bash
./run.sh server
```

This starts the camera server.

### On the Host Machine

Run:

```bash
./run.sh client
```

This starts the client and web server.

Once the client is running, open the address shown in the terminal in a web browser.

By default, it should be:

```text
http://localhost:8000
```

## Requirements

- Proper LAN connection

### StereoPi Compute Module

* Proper LAN connection to the host machine
* Python 3

### Host Machine

* Go **1.27.0**

## Repository Structure

```text
.
├── client/
├── server/
├── run.sh
└── README.md
```

The **server runs on the StereoPi**, while the **client runs on the host machine**. Both communicate over the LAN.

