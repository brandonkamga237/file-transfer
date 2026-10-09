# Kross server

## Features

- **Listening socket**: open a port and accept incoming TCP connections.
- **HTTP request parsing**: read the request line (method, path, version) and the headers.
- **Routing**: map each requested path to a resource or a handler.
- **Static file serving**: read a file from disk and return it with the correct MIME type.
- **Response building**: write the status line, headers, and body.
- **Error handling**: return 404 for a missing file and 500 for an internal error.
- **Concurrency**: handle several clients at the same time (threads, processes, or an event loop).
- **Connection handling**: manage keep-alive or close the connection cleanly.