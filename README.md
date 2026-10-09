# Kross

Kross is a lightweight web server written in Go, built directly on top of TCP sockets.

> Kross is in early development. Behavior and options may change between versions.

## Requirements

- Go 1.27 or later
- Linux or macOS (on Windows, use WSL)

## Installation

```sh
git clone https://github.com/brandonkamga237/kross.git
cd kross
make build
```

The binary is written to `bin/webserver`.

## Usage

```sh
./bin/webserver
```

By default the server listens on `127.0.0.1:8080`. Stop it with `Ctrl+C`.

| Flag       | Default     | Description                    |
|------------|-------------|--------------------------------|
| `-host`    | `127.0.0.1` | IPv4 address to listen on      |
| `-port`    | `8080`      | TCP port (0 picks a free port) |
| `-backlog` | `128`       | Pending connection queue size  |

```sh
./bin/webserver -host 0.0.0.0 -port 9000
```

## Development

```sh
make test   # run the test suite with the race detector
make vet    # run static checks
make run    # start the server without building a binary
```

## Contributing

Issues and pull requests are welcome. Please run `make vet` and `make test` before submitting a change.
