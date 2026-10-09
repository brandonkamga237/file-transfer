# Kross

Kross is a web server written in Go from the ground up, as a learning project.
It avoids `net/http` and builds each layer by hand, starting with a raw TCP socket.

## Status

The listening socket works: the server binds to an IPv4 address, listens with a
backlog you can set, accepts connections and shuts down cleanly on `SIGINT` or
`SIGTERM`. Accepted connections are closed right away for now. Reading HTTP
requests comes next.

## Requirements

- Go 1.27 or later
- A Unix system (Linux or macOS). The TCP listener uses the `syscall` package
  and is built only with the `unix` build tag. On Windows, use WSL.

## Getting started

```sh
make build              # builds bin/webserver
./bin/webserver         # listens on 127.0.0.1:8080
```

You can also start it without building:

```sh
make run
```

### Command-line flags

| Flag       | Default     | Description                    |
|------------|-------------|--------------------------------|
| `-host`    | `127.0.0.1` | IPv4 address to listen on      |
| `-port`    | `8080`      | TCP port (0 picks a free port) |
| `-backlog` | `128`       | Pending connection queue size  |

Example:

```sh
./bin/webserver -host 0.0.0.0 -port 9000 -backlog 256
```

## Development

```sh
make test   # go test -race ./...
make vet    # go vet ./...
```

## Project layout

```
cmd/webserver/        entrypoint: flags, signals, wiring
internal/config/      server settings and validation
internal/listener/    Listener interface and the syscall-based TCP listener
internal/conn/        ClientConnection, an io.ReadWriteCloser around net.Conn
internal/server/      lifecycle: bind, listen, accept loop, stop
```

The `net.Listen` function does not let you set the backlog, so `TcpListener`
creates the socket with system calls (`socket`, `setsockopt`, `bind`, `listen`)
and then converts it into a `net.Listener`.

## Roadmap

- [x] **Listening socket**: open a port and accept incoming TCP connections.
- [ ] **HTTP request parsing**: read the request line (method, path, version) and the headers.
- [ ] **Routing**: map each requested path to a resource or a handler.
- [ ] **Static file serving**: read a file from disk and return it with the correct MIME type.
- [ ] **Response building**: write the status line, headers and body.
- [ ] **Error handling**: return 404 for a missing file and 500 for an internal error.
- [ ] **Concurrency**: handle several clients at the same time.
- [ ] **Connection handling**: manage keep-alive or close the connection cleanly.
